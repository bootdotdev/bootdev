package render

import (
	"strings"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/messages"
)

func TestStepDetailsKeepLabelsSeparateFromValues(t *testing.T) {
	for _, tt := range []struct {
		name    string
		message messages.StartStepMsg
		label   string
		value   string
	}{
		{name: "command containing label text", message: messages.StartStepMsg{CMD: "Request: literal command"}, label: "Command", value: "Request: literal command"},
		{name: "request", message: messages.StartStepMsg{Method: "GET", URL: "http://localhost/Command:"}, label: "Request", value: "GET http://localhost/Command:"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			model := initModel(false, false)
			updated, _ := model.Update(tt.message)
			model = updated.(rootModel)
			model.finalized = true
			view := model.View()
			if want := "\n" + tt.label + ": " + tt.value + "\n"; !strings.Contains(view, want) {
				t.Fatalf("missing detail %q in:\n%s", want, view)
			}
		})
	}
}

func TestDetailedStepGroupsFailureAndDiagnostics(t *testing.T) {
	passed, failed := true, false
	expectedExit := 0
	step := stepModel{
		description: "Read log", detailLabel: "Command", detail: "cat repl.log",
		passed: &failed,
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
	view := renderDetailedStep(step, 1, true, nil)
	for _, expected := range []string{
		"STEP 1 · Read log\nCommand: cat repl.log",
		"  Checks:",
		"    FAIL · 2. Exit code\n      Expected: 0\n      Actual:   1",
		"  Standard error\n    missing file\n    check path",
		"  Standard output\n    <empty>",
		"    3. Expect stdout to contain:\n      greeting",
	} {
		if !strings.Contains(view, expected) {
			t.Errorf("missing %q in:\n%s", expected, view)
		}
	}
	previous := -1
	for _, marker := range []string{"PASS · 1. earlier passing check", "FAIL · 2. Exit code", "3. Expect stdout", "Standard output"} {
		position := strings.Index(view, marker)
		if position <= previous {
			t.Fatalf("%q missing or out of order in:\n%s", marker, view)
		}
		previous = position
	}
	if strings.Contains(view, "Command exit code") {
		t.Errorf("duplicate exit-code diagnostic in:\n%s", view)
	}
	if strings.Contains(view, "FAIL · STEP") {
		t.Errorf("expanded step heading repeats check status:\n%s", view)
	}
}

func TestOnlySubmitShowsCheckDetails(t *testing.T) {
	passed := true
	expectedStatus := 200
	step := stepModel{
		description: "Health", detailLabel: "Request", detail: "GET /health",
		tests: []testModel{{text: "Expecting status code: 200", passed: &passed}},
		result: &api.CLIStepResult{HTTPRequestResult: &api.HTTPRequestResult{
			StatusCode: 200,
			Request:    api.CLIStepHTTPRequest{Tests: []api.HTTPRequestTest{{StatusCode: &expectedStatus}}},
		}},
	}
	runOutput := renderDetailedStep(step, 1, false, nil)
	submitOutput := renderDetailedStep(step, 1, true, nil)
	if strings.Contains(runOutput, "Expected:") || strings.Contains(runOutput, "Expecting") || strings.Contains(runOutput, "PASS") {
		t.Fatalf("run should not display checks: %s", runOutput)
	}
	if !strings.Contains(runOutput, "Response status: 200") {
		t.Fatalf("run is missing response status: %s", runOutput)
	}
	if !strings.Contains(submitOutput, "1. HTTP status\n      Expected: 200\n      Actual:   200") {
		t.Fatalf("submit is missing expected/actual values: %s", submitOutput)
	}
}

func TestRunRetainsCommandOutputAndExitCode(t *testing.T) {
	expectedExit := 0
	step := stepModel{
		description: "Run", detailLabel: "Command", detail: "./app",
		tests: []testModel{{text: "Expect exit code 0"}},
		result: &api.CLIStepResult{CLICommandResult: &api.CLICommandResult{
			ExitCode: 1, Stdout: "hello", Stderr: "problem",
			Command: api.CLIStepCLICommand{Tests: []api.CLICommandTest{{ExitCode: &expectedExit}}},
		}},
	}
	view := renderDetailedStep(step, 1, false, nil)
	for _, expected := range []string{"Command: ./app", "Command exit code: 1", "hello", "problem"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("run missing %q: %s", expected, view)
		}
	}
	if strings.Contains(view, "Expect") {
		t.Fatalf("run includes checks: %s", view)
	}
}

func TestGradingErrorsUseVisualLimitsAtCheckAndStepLevel(t *testing.T) {
	failed := false
	for _, message := range []string{strings.Repeat("界", 20000), strings.Repeat("diagnostic\n", 100)} {
		for _, testIndex := range []int{0, -1, 2} {
			step := stepModel{
				description: "Example", passed: &failed, finished: true,
				tests: []testModel{{text: "Requirement", passed: &failed, finished: true}},
			}
			failure := &api.StructuredErrCLI{FailedStepIndex: 0, FailedTestIndex: testIndex, ErrorMessage: message}
			got := renderDetailedStep(step, 1, true, failure)
			if strings.Count(got, "Error:") != 1 || strings.Count(got, "... output visually truncated") != 1 {
				t.Fatalf("testIndex=%d: expected one truncated error", testIndex)
			}
			if strings.Contains(got, message) || strings.Contains(got, strings.Repeat("界", 5121)) || strings.Count(got, "diagnostic") > 32 {
				t.Fatalf("testIndex=%d: error exceeds visual limits", testIndex)
			}
			if otherStep := renderDetailedStep(step, 2, true, failure); strings.Contains(otherStep, "Error:") {
				t.Fatal("grading error displayed on the wrong step")
			}
			if run := renderDetailedStep(step, 1, false, failure); strings.Contains(run, "Error:") {
				t.Fatal("run exposed grading error")
			}
		}
	}
}
