package checks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	api "github.com/bootdotdev/bootdev/client"
)

const (
	cliCommandTimeout          = 5 * time.Minute
	maxCLIOutputBytesPerStream = 1024 * 1024
	commandWaitDelay           = 2 * time.Second
)

var errCLIOutputLimitExceeded = errors.New("CLI command output limit exceeded")

type boundedBuffer struct {
	buffer     bytes.Buffer
	limit      int
	truncated  bool
	onTruncate func()
}

func newBoundedBuffer(limit int, onTruncate func()) *boundedBuffer {
	return &boundedBuffer{
		limit:      max(limit, 0),
		onTruncate: onTruncate,
	}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	originalLength := len(p)
	remaining := max(b.limit-b.buffer.Len(), 0)
	toWrite := min(len(p), remaining)
	if toWrite > 0 {
		_, _ = b.buffer.Write(p[:toWrite])
	}
	if toWrite < len(p) && !b.truncated {
		b.truncated = true
		if b.onTruncate != nil {
			b.onTruncate()
		}
	}
	return originalLength, nil
}

func (b *boundedBuffer) String() string {
	return b.buffer.String()
}

func runCLICommand(command api.CLIStepCLICommand, variables map[string]string) (result api.CLICommandResult) {
	return runCLICommandWithLimits(command, variables, cliCommandTimeout, maxCLIOutputBytesPerStream)
}

func runCLICommandWithLimits(
	command api.CLIStepCLICommand,
	variables map[string]string,
	timeout time.Duration,
	maxOutputBytesPerStream int,
) (result api.CLICommandResult) {
	finalCommand := InterpolateVariables(command.Command, variables)
	result.FinalCommand = finalCommand
	result.Command = command

	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), timeout)
	defer cancelTimeout()
	ctx, cancelCommand := context.WithCancelCause(timeoutCtx)
	defer cancelCommand(nil)

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-Command", finalCommand)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", finalCommand)
	}

	cmd.Env = append(os.Environ(), "LANG=en_US.UTF-8")
	cmd.WaitDelay = commandWaitDelay
	cancelForOutputLimit := func() {
		cancelCommand(errCLIOutputLimitExceeded)
	}
	stdout := newBoundedBuffer(maxOutputBytesPerStream, cancelForOutputLimit)
	stderr := newBoundedBuffer(maxOutputBytesPerStream, cancelForOutputLimit)
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

	switch {
	case errors.Is(context.Cause(ctx), errCLIOutputLimitExceeded):
		result.Err = fmt.Sprintf("command output exceeded the %d-byte per-stream limit", maxOutputBytesPerStream)
		result.ExitCode = -2
	case errors.Is(context.Cause(ctx), context.DeadlineExceeded):
		result.Err = fmt.Sprintf("command timed out after %s", timeout)
		result.ExitCode = -2
	}

	if result.Err == "" {
		if err := parseStdoutVariables(result.Stdout, command.StdoutVariables, variables); err != nil {
			result.Err = err.Error()
		}
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
