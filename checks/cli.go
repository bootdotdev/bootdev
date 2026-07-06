package checks

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	api "github.com/bootdotdev/bootdev/client"
)

func runCLICommand(command api.CLIStepCLICommand, variables map[string]string) (result api.CLICommandResult) {
	finalCommand := InterpolateVariables(command.Command, variables)
	result.FinalCommand = finalCommand
	result.Command = command

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-Command", finalCommand)
	} else {
		cmd = exec.Command("sh", "-c", finalCommand)
	}

	cmd.Env = append(os.Environ(), "LANG=en_US.UTF-8")
	b, err := cmd.CombinedOutput()
	if ee, ok := err.(*exec.ExitError); ok {
		result.ExitCode = ee.ExitCode()
	} else if err != nil {
		result.ExitCode = -2
	}
	result.Stdout = strings.TrimRight(string(b), " \n\t\r")
	if command.StdoutFilterTmdl != nil {
		result.Stdout = ExtractTmdlBlock(result.Stdout, *command.StdoutFilterTmdl)
	}
	if err := parseStdoutVariables(result.Stdout, command.StdoutVariables, variables); err != nil {
		result.Err = err.Error()
	}
	result.Variables = maps.Clone(variables)
	return result
}

func parseStdoutVariables(stdout string, vardefs []api.CLICommandStdoutVariable, variables map[string]string) error {
	for _, vardef := range vardefs {
		if vardef.Name == "" {
			return fmt.Errorf("invalid stdout variable configuration")
		}
		if vardef.Regex == "" {
			return fmt.Errorf("invalid stdout variable configuration")
		}
		re, err := regexp.Compile(vardef.Regex)
		if err != nil {
			return fmt.Errorf("invalid stdout variable configuration")
		}
		if re.NumSubexp() != 1 {
			return fmt.Errorf("invalid stdout variable configuration")
		}

		matches := re.FindStringSubmatch(stdout)
		if len(matches) == 2 {
			variables[vardef.Name] = matches[1]
		}
	}

	return nil
}

func prettyPrintCLICommand(test api.CLICommandTest, variables map[string]string) string {
	if test.ExitCode != nil {
		return fmt.Sprintf("Expect exit code %d", *test.ExitCode)
	}
	if test.StdoutLinesGt != nil {
		return fmt.Sprintf("Expect > %d lines on stdout", *test.StdoutLinesGt)
	}
	if test.StdoutContainsAll != nil {
		var str strings.Builder
		str.WriteString("Expect stdout to contain all of:")
		for _, contains := range test.StdoutContainsAll {
			interpolatedContains := InterpolateVariables(contains, variables)
			fmt.Fprintf(&str, "\n      - '%s'", interpolatedContains)
		}
		return str.String()
	}
	if test.StdoutContainsNone != nil {
		var str strings.Builder
		str.WriteString("Expect stdout to contain none of:")
		for _, containsNone := range test.StdoutContainsNone {
			interpolatedContainsNone := InterpolateVariables(containsNone, variables)
			fmt.Fprintf(&str, "\n      - '%s'", interpolatedContainsNone)
		}
		return str.String()
	}
	if test.StdoutJq != nil {
		return prettyPrintStdoutJqTest(*test.StdoutJq, variables)
	}
	return ""
}
