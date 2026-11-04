package render

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
)

func TestCompactViewHidesSuccessfulDetailsAndExpandsFailure(t *testing.T) {
	passed := true
	failed := false
	m := initModel(true, false)
	m.finalized = true
	m.result = api.VerificationResultSlugFailure
	m.failure = &api.StructuredErrCLI{FailedStepIndex: 1, FailedTestIndex: 0, ErrorMessage: "expected failure"}
	m.steps = []stepModel{
		{
			description: "The health check succeeds",
			detailLabel: "Request",
			detail:      "GET http://localhost:3000/health",
			passed:      &passed,
			finished:    true,
			tests:       []testModel{{text: "Expecting status code: 200", passed: &passed, finished: true}},
			result: &api.CLIStepResult{HTTPRequestResult: &api.HTTPRequestResult{
				StatusCode: 200,
				BodyString: "successful response body",
			}},
		},
		{
			description: "Email changes require a password",
			detailLabel: "Request",
			detail:      "POST http://localhost:3000/account/email",
			passed:      &failed,
			finished:    true,
			tests:       []testModel{{text: "Expecting status code: 403", passed: &failed, finished: true}},
			result: &api.CLIStepResult{HTTPRequestResult: &api.HTTPRequestResult{
				StatusCode: 302,
				BodyString: "failed response body",
			}},
		},
		{
			description: "A later check",
			detailLabel: "Request",
			detail:      "GET http://localhost:3000/later",
			finished:    true,
		},
	}

	view := m.View()
	for _, expected := range []string{
		"PASS  Step 1 · The health check succeeds",
		"STEP 2 · Email changes require a password",
		"Request: POST http://localhost:3000/account/email",
		"Expecting status code: 403",
		"failed response body",
	} {
		if !strings.Contains(view, expected) {
			t.Errorf("view missing %q\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{
		"successful response body",
		"Expecting status code: 200",
		"A later check",
	} {
		if strings.Contains(view, unexpected) {
			t.Errorf("view unexpectedly contains %q\n%s", unexpected, view)
		}
	}
}

func TestNumberedStepsAcrossOutputModes(t *testing.T) {
	for _, tt := range []struct {
		name      string
		submit    bool
		verbose   bool
		finalized bool
	}{
		{name: "running"},
		{name: "run", finalized: true},
		{name: "submit", submit: true, finalized: true},
		{name: "verbose submit", submit: true, verbose: true, finalized: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := initModel(tt.submit, tt.verbose)
			m.finalized = tt.finalized
			m.steps = []stepModel{{description: "Build", finished: true}, {description: "Run", finished: true}}
			for range 2 {
				view := m.View()
				for _, expected := range []string{"1 · Build", "2 · Run"} {
					if strings.Count(view, expected) != 1 {
						t.Fatalf("expected one %q in view:\n%s", expected, view)
					}
				}
			}
		})
	}
}

func TestFailedViewOmitsSummaryAndPreservesSafeStepMessage(t *testing.T) {
	for _, tt := range []struct {
		name        string
		failure     *api.StructuredErrCLI
		detail      string
		detailLabel string
		safe        bool
	}{
		{name: "command", failure: &api.StructuredErrCLI{FailedStepIndex: 1, ErrorMessage: "wrong exit code"}, detailLabel: "Command", detail: "./app"},
		{name: "request", failure: &api.StructuredErrCLI{FailedStepIndex: 1, ErrorMessage: "wrong status"}, detailLabel: "Request", detail: "GET http://localhost/health"},
		{name: "safe step", failure: &api.StructuredErrCLI{FailedStepIndex: 1, ErrorMessage: "missing configuration"}, detailLabel: "Command", detail: "./app", safe: true},
		{name: "missing failure"},
		{name: "missing safe failure", safe: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := initModel(true, false)
			m.finalized = true
			m.result = api.VerificationResultSlugFailure
			if tt.safe {
				m.result = api.VerificationResultSlugNoop
			}
			m.failure = tt.failure
			failed := false
			m.steps = []stepModel{{description: "Build", finished: true}, {description: "Run", detailLabel: tt.detailLabel, detail: tt.detail, finished: true, passed: &failed}}
			view := m.View()
			if strings.Contains(view, "FAILED · ") {
				t.Fatalf("unexpected failure recap:\n%s", view)
			}
			if strings.Contains(view, "This was a safe step.") != tt.safe {
				t.Fatalf("incorrect safe-step message:\n%s", view)
			}
			if tt.detail != "" && strings.Count(view, tt.detail) != 1 {
				t.Fatalf("expected failed-step detail exactly once: %s", view)
			}
		})
	}
}

func TestCompactViewDoesNotHideStepsForInvalidFailureIndex(t *testing.T) {
	for _, failedStepIndex := range []int{-1, 2} {
		t.Run(fmt.Sprintf("index_%d", failedStepIndex), func(t *testing.T) {
			m := initModel(true, false)
			m.finalized = true
			m.result = api.VerificationResultSlugFailure
			m.failure = &api.StructuredErrCLI{FailedStepIndex: failedStepIndex}
			m.steps = []stepModel{
				{description: "The first step", finished: true},
				{description: "The second step", finished: true},
			}

			view := m.View()
			for _, description := range []string{"The first step", "The second step"} {
				if !strings.Contains(view, description) {
					t.Errorf("view missing %q for invalid failure index %d\n%s", description, failedStepIndex, view)
				}
			}
		})
	}
}

func TestSystemErrorViewDoesNotShowStepsAsPassed(t *testing.T) {
	m := initModel(true, false)
	m.finalized = true
	m.result = api.VerificationResultSlugSystemError
	m.steps = []stepModel{{
		description: "The command prints a greeting",
		finished:    true,
	}}

	view := m.View()
	for _, expected := range []string{
		"?  Step 1 · The command prints a greeting",
		"system error",
	} {
		if !strings.Contains(view, expected) {
			t.Errorf("view missing %q\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"PASS  Step 1 · The command prints a greeting", "All tests passed!"} {
		if strings.Contains(view, unexpected) {
			t.Errorf("view unexpectedly contains %q\n%s", unexpected, view)
		}
	}
}

func TestTruncateVisualOutputCapsLinesAndRunes(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{name: "many lines", output: strings.Repeat("line\n", 40)},
		{name: "long ASCII line", output: strings.Repeat("x", 6000)},
		{name: "long Unicode line", output: strings.Repeat("界", 6000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateVisualOutput(tt.output)
			if !strings.HasSuffix(got, "... output visually truncated") {
				t.Fatalf("expected truncation marker")
			}
			if strings.Count(got, "\n") > 32 {
				t.Fatalf("truncated output has too many lines: %d", strings.Count(got, "\n")+1)
			}
			if utf8.RuneCountInString(got) > 5120+1+utf8.RuneCountInString("... output visually truncated") {
				t.Fatalf("truncated output exceeds the rune limit")
			}
		})
	}
}

func TestHTTPRequestBodyUsesVisualOutputLimit(t *testing.T) {
	got := printHTTPRequestResult(api.HTTPRequestResult{
		StatusCode: 200,
		BodyString: strings.Repeat("x", 6000),
	})

	if !strings.Contains(got, "... output visually truncated") {
		t.Fatalf("expected HTTP response body to be visually truncated")
	}
}

func TestGradingExplanationStaysWithFailedCheck(t *testing.T) {
	expectedExit, expectedLines, expectedStatus := 0, 50, 200
	expectedString := "ready"
	for _, tt := range []struct {
		name      string
		step      api.CLIStep
		result    api.CLIStepResult
		wantError bool
	}{
		{
			name:      "truncated line count",
			step:      api.CLIStep{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{StdoutLinesGT: &expectedLines}}}},
			result:    api.CLIStepResult{CLICommandResult: &api.CLICommandResult{Stdout: strings.TrimSuffix(strings.Repeat("line\n", 40), "\n")}},
			wantError: true,
		},
		{
			name:      "compound command check",
			step:      api.CLIStep{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{ExitCode: &expectedExit, StdoutContainsAll: []string{"ready"}}}}},
			result:    api.CLIStepResult{CLICommandResult: &api.CLICommandResult{Stdout: "pending"}},
			wantError: true,
		},
		{
			name:   "simple exit code",
			step:   api.CLIStep{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{ExitCode: &expectedExit}}}},
			result: api.CLIStepResult{CLICommandResult: &api.CLICommandResult{ExitCode: 2}},
		},
		{
			name:   "simple HTTP status",
			step:   api.CLIStep{HTTPRequest: &api.CLIStepHTTPRequest{Tests: []api.HTTPRequestTest{{StatusCode: &expectedStatus}}}},
			result: api.CLIStepResult{HTTPRequestResult: &api.HTTPRequestResult{StatusCode: 500}},
		},
		{
			name: "JSON wrong type",
			step: api.CLIStep{HTTPRequest: &api.CLIStepHTTPRequest{Tests: []api.HTTPRequestTest{{JSONValue: &api.HTTPRequestTestJSONValue{
				Path: ".value", Operator: api.OpEquals, StringValue: &expectedString,
			}}}}},
			result:    api.CLIStepResult{HTTPRequestResult: &api.HTTPRequestResult{BodyString: `{"value":123}`}},
			wantError: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.CLICommandResult != nil {
				tt.result.CLICommandResult.Command = *tt.step.CLICommand
			} else {
				tt.result.HTTPRequestResult.Request = *tt.step.HTTPRequest
			}
			failure := checks.EvaluateCLIResults(api.CLIData{Steps: []api.CLIStep{tt.step}}, []api.CLIStepResult{tt.result})
			if failure == nil || failure.FailedTestIndex != 0 {
				t.Fatalf("expected check failure, got %+v", failure)
			}
			for _, submit := range []bool{false, true} {
				for _, verbose := range []bool{false, true} {
					failed := false
					m := initModel(submit, verbose)
					m.finalized = true
					m.result = api.VerificationResultSlugFailure
					m.failure = failure
					m.steps = []stepModel{{description: "Example", passed: &failed, finished: true, result: &tt.result,
						tests: []testModel{{text: "Requirement", passed: &failed, finished: true}}}}
					got := m.View()
					wantCount := 0
					if submit && tt.wantError {
						wantCount = 1
					}
					if strings.Count(got, "Error: "+failure.ErrorMessage) != wantCount || strings.Contains(got, "FAILED ·") {
						t.Fatalf("submit=%v verbose=%v: wrong grading diagnostic:\n%s", submit, verbose, got)
					}
					if wantCount == 1 && !strings.Contains(got, "\n      Error: "+failure.ErrorMessage) {
						t.Fatalf("grading error should be nested beneath the check:\n%s", got)
					}
					if tt.name == "truncated line count" && submit &&
						(!strings.Contains(got, "got 40") || !strings.Contains(got, "... output visually truncated")) {
						t.Fatalf("lost count with truncated output:\n%s", got)
					}
				}
			}
		})
	}
}

func TestCommandCaptureErrorAppearsOnlyInSubmittedStep(t *testing.T) {
	failed := false
	expectedExit := 0
	const captureError = "missing value for variable 'token'"
	for _, tt := range []struct {
		name    string
		submit  bool
		verbose bool
	}{
		{name: "run"},
		{name: "expanded run", verbose: true},
		{name: "submit", submit: true},
		{name: "expanded submit", submit: true, verbose: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := initModel(tt.submit, tt.verbose)
			m.finalized = true
			if tt.submit {
				m.result = api.VerificationResultSlugFailure
				m.failure = &api.StructuredErrCLI{FailedStepIndex: 0, FailedTestIndex: -1, ErrorMessage: captureError}
			}
			m.steps = []stepModel{{
				description: "Capture token", detailLabel: "Command", detail: "printf hello", finished: true, passed: &failed,
				tests: []testModel{{text: "Expect exit code 0", finished: true}},
				result: &api.CLIStepResult{CLICommandResult: &api.CLICommandResult{
					Err: captureError, ExitCode: 0, Stdout: "hello",
					Command: api.CLIStepCLICommand{Tests: []api.CLICommandTest{{ExitCode: &expectedExit}}},
				}},
			}}
			got := m.View()
			expectedCount := 0
			if tt.submit {
				expectedCount = 1
			}
			if strings.Count(got, "Error: "+captureError) != expectedCount {
				t.Fatalf("unexpected capture error visibility:\n%s", got)
			}
			if tt.submit && strings.Index(got, captureError) > strings.Index(got, "Standard output") {
				t.Fatalf("error should appear inside the step before its output:\n%s", got)
			}
			if !strings.Contains(got, "Command exit code: 0") || !strings.Contains(got, "hello") || strings.Contains(got, "FAILED ·") {
				t.Fatalf("lost command diagnostics or restored footer:\n%s", got)
			}
		})
	}
}

func TestCaptureValidationFailureAppearsOnSubmittedStep(t *testing.T) {
	expectedExit, expectedStatus := 0, 200
	for _, tt := range []struct {
		name   string
		step   api.CLIStep
		result api.CLIStepResult
	}{
		{
			name: "command",
			step: api.CLIStep{CLICommand: &api.CLIStepCLICommand{
				Tests:           []api.CLICommandTest{{ExitCode: &expectedExit}},
				StdoutVariables: []api.CLICommandStdoutVariable{{Name: "token", Regex: "token=(.*)"}},
			}},
			result: api.CLIStepResult{CLICommandResult: &api.CLICommandResult{
				Stdout: "token=expected", Variables: map[string]string{"token": "different"},
			}},
		},
		{
			name: "HTTP",
			step: api.CLIStep{HTTPRequest: &api.CLIStepHTTPRequest{
				Tests:             []api.HTTPRequestTest{{StatusCode: &expectedStatus}},
				ResponseVariables: []api.HTTPRequestResponseVariable{{Name: "token", Path: ".token"}},
			}},
			result: api.CLIStepResult{HTTPRequestResult: &api.HTTPRequestResult{
				StatusCode: 200, BodyString: `{"token":"expected"}`, Variables: map[string]string{"token": "different"},
			}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.CLICommandResult != nil {
				tt.result.CLICommandResult.Command = *tt.step.CLICommand
			} else {
				tt.result.HTTPRequestResult.Request = *tt.step.HTTPRequest
			}
			failure := checks.EvaluateCLIResults(api.CLIData{Steps: []api.CLIStep{tt.step}}, []api.CLIStepResult{tt.result})
			if failure == nil || failure.FailedTestIndex <= 0 {
				t.Fatalf("expected capture validation failure outside check list, got %+v", failure)
			}
			for _, submit := range []bool{true, false} {
				for _, verbose := range []bool{true, false} {
					passed, failed := true, false
					m := initModel(submit, verbose)
					m.finalized = true
					m.result = api.VerificationResultSlugFailure
					m.failure = failure
					m.steps = []stepModel{{description: "Capture", finished: true, passed: &failed, result: &tt.result,
						tests: []testModel{{text: "Requirement", passed: &passed, finished: true}}}}
					got := m.View()
					wantCount := 0
					if submit {
						wantCount = 1
					}
					if strings.Count(got, "Error: "+failure.ErrorMessage) != wantCount {
						t.Fatalf("submit=%v verbose=%v: unexpected capture diagnostic:\n%s", submit, verbose, got)
					}
					if submit && !strings.Contains(got, "\n  Error: "+failure.ErrorMessage) {
						t.Fatalf("capture diagnostic is not at step level:\n%s", got)
					}
				}
			}
		})
	}
}
