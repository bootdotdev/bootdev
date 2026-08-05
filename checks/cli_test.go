package checks

import (
	"runtime"
	"strings"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
)

func TestRunCLICommandCapsOutput(t *testing.T) {
	command := `printf 'abcdefgh'`
	if runtime.GOOS == "windows" {
		command = `[Console]::Out.Write('abcdefgh')`
	}

	variables := map[string]string{}
	result := runCLICommandWithOutputLimit(
		api.CLIStepCLICommand{
			Command: command,
			StdoutVariables: []api.CLICommandStdoutVariable{{
				Name:  "partial",
				Regex: `(abcd)`,
			}},
		},
		variables,
		4,
	)

	if !strings.Contains(result.Err, "per-stream limit") {
		t.Fatalf("command error = %q, want per-stream output limit error", result.Err)
	}
	if result.ExitCode >= 0 {
		t.Fatalf("exit code = %d, want internal failure", result.ExitCode)
	}
	if result.Stdout != "abcd" {
		t.Fatalf("stdout = %q, want capped output %q", result.Stdout, "abcd")
	}
	if _, ok := variables["partial"]; ok {
		t.Fatal("truncated output unexpectedly populated a stdout variable")
	}
}

func TestRunCLICommandCapturesStdoutVariables(t *testing.T) {
	variables := map[string]string{}
	result := runCLICommand(api.CLIStepCLICommand{
		Command: `go env GOOS`,
		StdoutVariables: []api.CLICommandStdoutVariable{{
			Name:  "goos",
			Regex: `([a-z0-9]+)`,
		}},
	}, variables)

	if result.Err != "" {
		t.Fatalf("unexpected command error: %s", result.Err)
	}
	if result.Variables["goos"] != runtime.GOOS {
		t.Fatalf("captured goos = %q, want %q", result.Variables["goos"], runtime.GOOS)
	}
	if variables["goos"] != runtime.GOOS {
		t.Fatalf("shared goos = %q, want %q", variables["goos"], runtime.GOOS)
	}
}

func TestRunCLICommandKeepsStderrSeparateFromStdoutChecks(t *testing.T) {
	command := `printf 'stdout-value\n'; printf 'stderr-value\n' >&2`
	if runtime.GOOS == "windows" {
		command = `Write-Output 'stdout-value'; [Console]::Error.WriteLine('stderr-value')`
	}

	variables := map[string]string{}
	step := api.CLIStepCLICommand{
		Command: command,
		StdoutVariables: []api.CLICommandStdoutVariable{{
			Name:  "stderr_value",
			Regex: `(stderr-value)`,
		}},
		Tests: []api.CLICommandTest{{
			StdoutContainsAll: []string{"stderr-value"},
		}},
	}

	result := runCLICommand(step, variables)

	if result.Stdout != "stdout-value" {
		t.Fatalf("stdout = %q, want stdout-value", result.Stdout)
	}
	if result.Stderr != "stderr-value" {
		t.Fatalf("stderr = %q, want stderr-value", result.Stderr)
	}
	if strings.Contains(result.Stdout, "stderr-value") {
		t.Fatalf("stdout unexpectedly contains stderr: %q", result.Stdout)
	}
	if _, ok := variables["stderr_value"]; ok {
		t.Fatalf("stderr unexpectedly populated a stdout variable")
	}
	if failure := evaluateCLICommandTests(0, step, result); failure == nil {
		t.Fatal("stderr unexpectedly satisfied a stdout check")
	}
}

func TestRunCLICommandInterpolatesCapturedStdoutVariables(t *testing.T) {
	variables := map[string]string{}

	first := runCLICommand(api.CLIStepCLICommand{
		Command: `go env -json GOOS`,
		StdoutVariables: []api.CLICommandStdoutVariable{{
			Name:  "goenv",
			Regex: `"([A-Z]+)"`,
		}},
	}, variables)
	if first.Err != "" {
		t.Fatalf("unexpected first command error: %s", first.Err)
	}

	second := runCLICommand(api.CLIStepCLICommand{
		Command: `go env ${goenv}`,
	}, variables)
	if second.Stdout != runtime.GOOS {
		t.Fatalf("second stdout = %q, want %q", second.Stdout, runtime.GOOS)
	}
}

func TestParseStdoutVariablesUsesGenericConfigurationError(t *testing.T) {
	tests := []struct {
		name   string
		vardef api.CLICommandStdoutVariable
	}{
		{
			name:   "missing name",
			vardef: api.CLICommandStdoutVariable{Regex: `token=([a-z0-9]+)`},
		},
		{
			name:   "missing regex",
			vardef: api.CLICommandStdoutVariable{Name: "token"},
		},
		{
			name:   "invalid regex",
			vardef: api.CLICommandStdoutVariable{Name: "token", Regex: `token=([a-z0-9]+`},
		},
		{
			name:   "too many capture groups",
			vardef: api.CLICommandStdoutVariable{Name: "token", Regex: `token=([a-z]+)([0-9]+)`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variables := map[string]string{}
			err := parseStdoutVariables("token=abc123", []api.CLICommandStdoutVariable{tt.vardef}, variables)
			if err == nil {
				t.Fatal("expected parse error")
			}
			if err.Error() != "invalid stdout variable configuration" {
				t.Fatalf("error = %q, want invalid stdout variable configuration", err.Error())
			}
		})
	}
}
