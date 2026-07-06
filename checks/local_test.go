package checks

import (
	"testing"

	api "github.com/bootdotdev/bootdev/client"
)

func TestLocalSubmissionEventPassesCLIAndHTTPResults(t *testing.T) {
	cliData := api.CLIData{Steps: []api.CLIStep{
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{
			{ExitCode: intPtr(0)},
			{StdoutContainsAll: []string{"hello ${name}"}},
		}}},
		{HTTPRequest: &api.CLIStepHTTPRequest{Tests: []api.HTTPRequestTest{
			{StatusCode: intPtr(200)},
			{HeadersContain: &api.HTTPRequestTestHeader{Key: "Set-Cookie", Value: "session_id="}},
			{JSONValue: &api.HTTPRequestTestJSONValue{
				Path:        ".app",
				Operator:    api.OpEquals,
				StringValue: stringPtr("bearly-secure"),
			}},
		}}},
	}}

	results := []api.CLIStepResult{
		{CLICommandResult: &api.CLICommandResult{
			ExitCode:  0,
			Stdout:    "hello Boots",
			Variables: map[string]string{"name": "Boots"},
		}},
		{HTTPRequestResult: &api.HTTPRequestResult{
			StatusCode:      200,
			ResponseHeaders: map[string]string{"Set-Cookie": "session_id=abc123; Path=/"},
			BodyString:      `{"app":"bearly-secure"}`,
			Variables:       map[string]string{},
		}},
	}

	event := LocalSubmissionEvent(cliData, results)
	if event.ResultSlug != api.VerificationResultSlugSuccess {
		t.Fatalf("ResultSlug = %q, want success; failure = %#v", event.ResultSlug, event.StructuredErrCLI)
	}
	if event.StructuredErrCLI != nil {
		t.Fatalf("unexpected failure: %#v", event.StructuredErrCLI)
	}
}

func TestLocalSubmissionEventReportsFirstFailure(t *testing.T) {
	cliData := api.CLIData{Steps: []api.CLIStep{
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{
			{ExitCode: intPtr(0)},
			{StdoutContainsAll: []string{"expected"}},
		}}},
	}}
	results := []api.CLIStepResult{
		{CLICommandResult: &api.CLICommandResult{
			ExitCode:  0,
			Stdout:    "actual",
			Variables: map[string]string{},
		}},
	}

	event := LocalSubmissionEvent(cliData, results)
	if event.ResultSlug != api.VerificationResultSlugFailure {
		t.Fatalf("ResultSlug = %q, want failure", event.ResultSlug)
	}
	if event.StructuredErrCLI == nil {
		t.Fatal("expected structured failure")
	}
	if event.StructuredErrCLI.FailedStepIndex != 0 || event.StructuredErrCLI.FailedTestIndex != 1 {
		t.Fatalf("failure = %#v, want step 0 test 1", event.StructuredErrCLI)
	}
}

func TestEvaluateCLICommandReportsStdoutVariableParseError(t *testing.T) {
	cliData := api.CLIData{Steps: []api.CLIStep{
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{
			{ExitCode: intPtr(0)},
		}}},
	}}
	results := []api.CLIStepResult{
		{CLICommandResult: &api.CLICommandResult{
			ExitCode: 0,
			Err:      "invalid stdout variable configuration",
		}},
	}

	event := LocalSubmissionEvent(cliData, results)
	if event.ResultSlug != api.VerificationResultSlugFailure {
		t.Fatalf("ResultSlug = %q, want failure", event.ResultSlug)
	}
	if event.StructuredErrCLI == nil {
		t.Fatal("expected structured failure")
	}
	if event.StructuredErrCLI.ErrorMessage != "invalid stdout variable configuration" {
		t.Fatalf("ErrorMessage = %q, want stdout variable error", event.StructuredErrCLI.ErrorMessage)
	}
}

func TestEvaluateStdoutJq(t *testing.T) {
	err := evaluateStdoutJq(
		"{\"ok\":true}",
		api.StdoutJqTest{
			InputMode: "json",
			Query:     ".ok",
			ExpectedResults: []api.JqExpectedResult{
				{Type: api.JqTypeBool, Operator: "==", Value: true},
			},
		},
		map[string]string{},
	)
	if err != nil {
		t.Fatalf("unexpected jq failure: %v", err)
	}
}

func TestValuesEqualPreservesTypes(t *testing.T) {
	tests := []struct {
		name string
		got  any
		want any
		ok   bool
	}{
		{name: "same strings", got: "1", want: "1", ok: true},
		{name: "string and int", got: "1", want: 1, ok: false},
		{name: "string and bool", got: "true", want: true, ok: false},
		{name: "same bools", got: true, want: true, ok: true},
		{name: "numeric int and float", got: 1, want: 1.0, ok: true},
		{name: "numeric json number and int", got: testJSONNumber("1"), want: 1, ok: true},
		{name: "nil and string", got: nil, want: "<nil>", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := valuesEqual(tt.got, tt.want); got != tt.ok {
				t.Fatalf("valuesEqual(%#v, %#v) = %v, want %v", tt.got, tt.want, got, tt.ok)
			}
		})
	}
}

type testJSONNumber string

func (n testJSONNumber) String() string {
	return string(n)
}

func intPtr(v int) *int {
	return &v
}

func stringPtr(v string) *string {
	return &v
}
