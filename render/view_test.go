package render

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

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
			detail:      "Request: GET http://localhost:3000/health",
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
			detail:      "Request: POST http://localhost:3000/account/email",
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
			detail:      "Request: GET http://localhost:3000/later",
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
			if m.steps[0].description != "Build" {
				t.Fatal("rendering changed the stored description")
			}
		})
	}
}

func TestFailureSummaryIncludesFailedStepContext(t *testing.T) {
	for _, tt := range []struct {
		name    string
		failure *api.StructuredErrCLI
		detail  string
		safe    bool
		want    string
	}{
		{name: "command", failure: &api.StructuredErrCLI{FailedStepIndex: 1, ErrorMessage: "wrong exit code"}, detail: "Command: ./app", want: "Step 2 · Run\nCommand: ./app\nError: wrong exit code"},
		{name: "request", failure: &api.StructuredErrCLI{FailedStepIndex: 1, ErrorMessage: "wrong status"}, detail: "Request: GET http://localhost/health", want: "Step 2 · Run\nRequest: GET http://localhost/health\nError: wrong status"},
		{name: "safe step", failure: &api.StructuredErrCLI{FailedStepIndex: 1, ErrorMessage: "missing configuration"}, detail: "Command: ./app", safe: true, want: "Step 2 · Run\nCommand: ./app\nError: missing configuration"},
		{name: "missing failure", want: "Step unknown\nError: unknown"},
		{name: "missing safe failure", safe: true, want: "Step unknown\nError: unknown"},
		{name: "negative index", failure: &api.StructuredErrCLI{FailedStepIndex: -1, ErrorMessage: "invalid"}, want: "Step unknown\nError: invalid"},
		{name: "out of range", failure: &api.StructuredErrCLI{FailedStepIndex: 2, ErrorMessage: "invalid"}, want: "Step unknown\nError: invalid"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := initModel(true, false)
			m.finalized = true
			m.result = api.VerificationResultSlugFailure
			if tt.safe {
				m.result = api.VerificationResultSlugNoop
			}
			m.failure = tt.failure
			m.steps = []stepModel{{description: "Build", finished: true}, {description: "Run", detail: tt.detail, finished: true}}
			view := m.View()
			_, recap, ok := strings.Cut(view, "FAILED · ")
			if !ok || !strings.Contains(recap, tt.want) {
				t.Fatalf("expected final recap containing %q, got:\n%s", tt.want, view)
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

func TestRunViewNeverShowsGradingResults(t *testing.T) {
	passed, failed := true, false
	m := initModel(false, false)
	m.finalized = true
	m.result = api.VerificationResultSlugFailure
	m.failure = &api.StructuredErrCLI{FailedStepIndex: 0, ErrorMessage: "grading failure"}
	m.steps = []stepModel{
		{description: "First command", finished: true, passed: &failed, tests: []testModel{{text: "Expect exit code 0", passed: &failed}}},
		{description: "Second command", finished: true, passed: &passed, tests: []testModel{{text: "Expect stdout to contain hello", passed: &passed}}},
	}
	view := m.View()
	for _, forbidden := range []string{"PASS", "FAIL", "grading failure", "Checks not evaluated", "Expect exit code 0", "Expect stdout to contain hello"} {
		if strings.Contains(view, forbidden) {
			t.Fatalf("run exposed grading text %q:\n%s", forbidden, view)
		}
	}
	for _, expected := range []string{"First command", "Second command"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("run lost %q:\n%s", expected, view)
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
