package checks

import (
	"runtime"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
)

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

func TestParseStdoutVariablesRequiresOneCaptureGroup(t *testing.T) {
	variables := map[string]string{}
	err := parseStdoutVariables("token=abc123", []api.CLICommandStdoutVariable{{
		Name:  "token",
		Regex: `token=([a-z]+)([0-9]+)`,
	}}, variables)

	if err == nil {
		t.Fatal("expected parse error")
	}
	if err.Error() != "invalid stdout variable configuration" {
		t.Fatalf("error = %q, want invalid stdout variable configuration", err.Error())
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
