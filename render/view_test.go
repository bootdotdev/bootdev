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
		"✓  The health check succeeds",
		"X  Email changes require a password",
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
		"?  The command prints a greeting",
		"system error",
	} {
		if !strings.Contains(view, expected) {
			t.Errorf("view missing %q\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"✓  The command prints a greeting", "All tests passed!"} {
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
