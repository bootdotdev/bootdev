package render

import (
	"strings"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
)

func TestDetailedStepGroupsFailureAndDiagnostics(t *testing.T) {
	passed, failed := true, false
	expectedExit := 0
	step := stepModel{
		description: "Read log", detail: "Command: cat repl.log",
		tests: []testModel{
			{text: "earlier passing check", passed: &passed},
			{text: "Expect exit code 0", passed: &failed},
			{text: "Expect stdout to contain:\n  greeting"},
		},
		result: &api.CLIStepResult{CLICommandResult: &api.CLICommandResult{
			ExitCode: 1, Stderr: "missing file\ncheck path",
			Command: api.CLIStepCLICommand{Tests: []api.CLICommandTest{{}, {ExitCode: &expectedExit}, {}}},
		}},
	}
	view := renderDetailedStep(step, 1, true)
	for _, expected := range []string{
		"STEP 1 · Read log\nCommand: cat repl.log",
		"  FAIL · Exit code\n    Expected: 0\n    Actual:   1",
		"  Standard error\n    missing file\n    check path",
		"  Checks not evaluated\n    Expect stdout to contain:\n      greeting",
	} {
		if !strings.Contains(view, expected) {
			t.Errorf("missing %q in:\n%s", expected, view)
		}
	}
	if strings.Index(view, "FAIL ·") > strings.Index(view, "PASS ·") {
		t.Error("passing check buried the failure")
	}
	if strings.Contains(view, "Standard output") || strings.Contains(view, "Command exit code") {
		t.Errorf("empty or duplicate diagnostics in:\n%s", view)
	}
}

func TestOnlySubmitShowsCheckDetails(t *testing.T) {
	passed := true
	expectedStatus := 200
	step := stepModel{
		description: "Health", detail: "Request: GET /health",
		tests: []testModel{{text: "Expecting status code: 200", passed: &passed}},
		result: &api.CLIStepResult{HTTPRequestResult: &api.HTTPRequestResult{
			StatusCode: 200,
			Request:    api.CLIStepHTTPRequest{Tests: []api.HTTPRequestTest{{StatusCode: &expectedStatus}}},
		}},
	}
	runOutput := renderDetailedStep(step, 1, false)
	submitOutput := renderDetailedStep(step, 1, true)
	if strings.Contains(runOutput, "Expected:") || strings.Contains(runOutput, "Expecting") || strings.Contains(runOutput, "PASS") {
		t.Fatalf("run should not display checks: %s", runOutput)
	}
	if !strings.Contains(runOutput, "Response status: 200") {
		t.Fatalf("run is missing response status: %s", runOutput)
	}
	if !strings.Contains(submitOutput, "HTTP status\n    Expected: 200\n    Actual:   200") {
		t.Fatalf("submit is missing expected/actual values: %s", submitOutput)
	}
}

func TestRunRetainsCommandOutputAndExitCode(t *testing.T) {
	expectedExit := 0
	step := stepModel{
		description: "Run", detail: "Command: ./app",
		tests: []testModel{{text: "Expect exit code 0"}},
		result: &api.CLIStepResult{CLICommandResult: &api.CLICommandResult{
			ExitCode: 1, Stdout: "hello", Stderr: "problem",
			Command: api.CLIStepCLICommand{Tests: []api.CLICommandTest{{ExitCode: &expectedExit}}},
		}},
	}
	view := renderDetailedStep(step, 1, false)
	for _, expected := range []string{"Command: ./app", "Command exit code: 1", "hello", "problem"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("run missing %q: %s", expected, view)
		}
	}
	if strings.Contains(view, "Expect") {
		t.Fatalf("run includes checks: %s", view)
	}
}
