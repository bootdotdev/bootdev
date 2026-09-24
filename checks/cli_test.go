package checks

import (
	"encoding/json"
	"runtime"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
)

func TestRunCLICommandCapsOutput(t *testing.T) {
	for _, tc := range []struct {
		name, command, windowsCommand, stdout, stderr, pattern, value string
		exitCode                                                      int
		wantFailure                                                   bool
	}{
		{
			name:           "stdout overflow",
			command:        `printf 'abcdefgh'`,
			windowsCommand: `[Console]::Out.Write('abcdefgh')`,
			stdout:         "abcd", pattern: `^(abcd)$`, value: "abcd",
		},
		{
			name:           "stderr overflow",
			command:        `printf 'ok'; printf 'abcdefgh' >&2`,
			windowsCommand: `[Console]::Out.Write('ok'); [Console]::Error.Write('abcdefgh')`,
			stdout:         "ok", stderr: "abcd", pattern: `^(ok)$`, value: "ok",
		},
		{
			name:           "nonzero exit",
			command:        `printf 'abcdefgh'; exit 7`,
			windowsCommand: `[Console]::Out.Write('abcdefgh'); exit 7`,
			stdout:         "abcd", pattern: `^(abcd)$`, value: "abcd", exitCode: 7,
		},
		{
			name:           "capture beyond limit",
			command:        `printf 'abcdefgh'`,
			windowsCommand: `[Console]::Out.Write('abcdefgh')`,
			stdout:         "abcd", pattern: `(efgh)`, wantFailure: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command := tc.command
			if runtime.GOOS == "windows" {
				command = tc.windowsCommand
			}
			step := api.CLIStepCLICommand{
				Command:         command,
				StdoutVariables: []api.CLICommandStdoutVariable{{Name: "token", Regex: tc.pattern}},
			}
			variables := map[string]string{"token": "old"}
			result := runCLICommandWithOutputLimit(step, variables, 4, defaultShell())
			if result.ExitCode != tc.exitCode {
				t.Fatalf("exit code = %d, want %d", result.ExitCode, tc.exitCode)
			}
			if result.Stdout != tc.stdout || result.Stderr != tc.stderr {
				t.Fatalf("stdout/stderr = %q/%q, want %q/%q", result.Stdout, result.Stderr, tc.stdout, tc.stderr)
			}
			if tc.wantFailure {
				if result.Err == "" {
					t.Fatal("missing capture should fail")
				}
				if _, found := variables["token"]; found {
					t.Fatal("failed capture retained the old token")
				}
				return
			}
			if result.Err != "" {
				t.Fatalf("unexpected command error: %s", result.Err)
			}
			if variables["token"] != tc.value {
				t.Fatalf("capture = %q, want %q", variables["token"], tc.value)
			}
		})
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

	result := runCLICommand(step, variables, defaultShell())

	if result.Stdout != "stdout-value" {
		t.Fatalf("stdout = %q, want stdout-value", result.Stdout)
	}
	if result.Stderr != "stderr-value" {
		t.Fatalf("stderr = %q, want stderr-value", result.Stderr)
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
	}, variables, defaultShell())
	if first.Err != "" {
		t.Fatalf("unexpected first command error: %s", first.Err)
	}

	if first.Variables["goenv"] != "GOOS" {
		t.Fatalf("captured variable = %q, want GOOS", first.Variables["goenv"])
	}

	second := runCLICommand(api.CLIStepCLICommand{
		Command: `go env ${goenv}`,
	}, variables, defaultShell())
	if second.Stdout != runtime.GOOS {
		t.Fatalf("second stdout = %q, want %q", second.Stdout, runtime.GOOS)
	}
}

func TestParseStdoutVariablesRejectsInvalidConfiguration(t *testing.T) {
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
		})
	}
}

func TestRunCLICommandPowerShellUTF8(t *testing.T) {
	shell := defaultShell()
	if runtime.GOOS != "windows" {
		var err error
		shell, err = resolveShell("pwsh")
		if err != nil {
			t.Skipf("PowerShell is unavailable: %v", err)
		}
	}

	result := runCLICommand(api.CLIStepCLICommand{
		Command: `'• Żółć' | ConvertTo-Json -Compress`,
	}, map[string]string{}, shell)
	if result.Err != "" || result.ExitCode != 0 {
		t.Fatalf("error = %q, exit code = %d, stderr = %q", result.Err, result.ExitCode, result.Stderr)
	}

	var got string
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatalf("invalid JSON output %q: %v", result.Stdout, err)
	}
	if got != "• Żółć" {
		t.Fatalf("decoded output = %q, want %q", got, "• Żółć")
	}
}
