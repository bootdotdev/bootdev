package checks

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	api "github.com/bootdotdev/bootdev/client"
)

const maxCLIOutputBytesPerStream = 1024 * 1024

type boundedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func newBoundedBuffer(limit int) *boundedBuffer {
	return &boundedBuffer{limit: max(limit, 0)}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	originalLength := len(p)

	remaining := max(b.limit-b.buffer.Len(), 0)
	toWrite := min(len(p), remaining)
	if toWrite > 0 {
		_, _ = b.buffer.Write(p[:toWrite])
	}
	if toWrite < len(p) {
		b.truncated = true
	}

	return originalLength, nil
}

func (b *boundedBuffer) String() string {
	return b.buffer.String()
}

func runCLICommand(command api.CLIStepCLICommand, variables map[string]string) (result api.CLICommandResult) {
	return runCLICommandWithOutputLimit(command, variables, maxCLIOutputBytesPerStream)
}

func runCLICommandWithOutputLimit(
	command api.CLIStepCLICommand,
	variables map[string]string,
	maxOutputBytesPerStream int,
) (result api.CLICommandResult) {
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
	stdout := newBoundedBuffer(maxOutputBytesPerStream)
	stderr := newBoundedBuffer(maxOutputBytesPerStream)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		result.ExitCode = ee.ExitCode()
	} else if err != nil {
		result.ExitCode = -2
	}

	result.Stdout = strings.TrimRight(stdout.String(), " \n\t\r")
	result.Stderr = strings.TrimRight(stderr.String(), " \n\t\r")
	if command.StdoutFilterTmdl != nil {
		result.Stdout = ExtractTmdlBlock(result.Stdout, *command.StdoutFilterTmdl)
	}

	if stdout.truncated || stderr.truncated {
		result.Err = fmt.Sprintf("command output exceeded the %d-byte per-stream limit", maxOutputBytesPerStream)
		result.ExitCode = -2
	} else if err := parseStdoutVariables(result.Stdout, command.StdoutVariables, variables); err != nil {
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

	if test.StdoutLinesGT != nil {
		return fmt.Sprintf("Expect > %d lines on stdout", *test.StdoutLinesGT)
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
