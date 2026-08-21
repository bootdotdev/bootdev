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
			{HeadersEqual: &api.HTTPRequestTestHeader{Key: "Set-Cookie", Value: "session_id=abc123; Path=/"}},
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

func TestEvaluateCLICommandReportsExecutionError(t *testing.T) {
	const message = "invalid stdout variable configuration"
	failure := evaluateCLICommandTests(
		0,
		api.CLIStepCLICommand{},
		api.CLICommandResult{Err: message},
	)

	if failure == nil {
		t.Fatal("expected structured failure")
	}
	if failure.ErrorMessage != message {
		t.Fatalf("ErrorMessage = %q, want %q", failure.ErrorMessage, message)
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

func TestEvaluateHTTPRequestTestsHeaderAndTrailerEquality(t *testing.T) {
	tests := []struct {
		name        string
		test        api.HTTPRequestTest
		result      api.HTTPRequestResult
		wantFailure bool
	}{
		{
			name: "header name is case insensitive",
			test: api.HTTPRequestTest{HeadersEqual: &api.HTTPRequestTestHeader{
				Key:   "X-Request-ID",
				Value: "abc123",
			}},
			result: api.HTTPRequestResult{
				ResponseHeaders: map[string]string{"x-request-id": "abc123"},
			},
		},
		{
			name: "header value is case sensitive",
			test: api.HTTPRequestTest{HeadersEqual: &api.HTTPRequestTestHeader{
				Key:   "X-Request-ID",
				Value: "abc123",
			}},
			result: api.HTTPRequestResult{
				ResponseHeaders: map[string]string{"X-Request-ID": "ABC123"},
			},
			wantFailure: true,
		},
		{
			name: "trailer name is case insensitive",
			test: api.HTTPRequestTest{TrailersEqual: &api.HTTPRequestTestHeader{
				Key:   "X-Checksum",
				Value: "sha256:abc",
			}},
			result: api.HTTPRequestResult{
				ResponseTrailers: map[string]string{"x-checksum": "sha256:abc"},
			},
		},
		{
			name: "trailer value is case sensitive",
			test: api.HTTPRequestTest{TrailersEqual: &api.HTTPRequestTestHeader{
				Key:   "X-Checksum",
				Value: "sha256:abc",
			}},
			result: api.HTTPRequestResult{
				ResponseTrailers: map[string]string{"X-Checksum": "SHA256:ABC"},
			},
			wantFailure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := api.CLIStepHTTPRequest{Tests: []api.HTTPRequestTest{tt.test}}
			failure := evaluateHTTPRequestTests(0, request, tt.result)
			if (failure != nil) != tt.wantFailure {
				t.Fatalf("failure = %#v, wantFailure = %t", failure, tt.wantFailure)
			}
		})
	}
}

func TestLocalSubmissionEventRejectsMissingHTTPResponseCaptures(t *testing.T) {
	tests := []struct {
		name    string
		request api.CLIStepHTTPRequest
		result  api.HTTPRequestResult
	}{
		{
			name: "response body variable",
			request: api.CLIStepHTTPRequest{
				Tests:             []api.HTTPRequestTest{{StatusCode: intPtr(200)}},
				ResponseVariables: []api.HTTPRequestResponseVariable{{Name: "token", Path: ".token"}},
			},
			result: api.HTTPRequestResult{
				StatusCode: 200,
				BodyString: `{"message":"missing token"}`,
				Variables:  map[string]string{"token": "stale-token"},
			},
		},
		{
			name: "response header variable",
			request: api.CLIStepHTTPRequest{
				Tests: []api.HTTPRequestTest{{StatusCode: intPtr(200)}},
				ResponseHeaderVariables: []api.HTTPRequestResponseHeaderVariable{{
					Name:   "requestID",
					Header: "X-Request-ID",
				}},
			},
			result: api.HTTPRequestResult{
				StatusCode:      200,
				ResponseHeaders: map[string]string{},
				Variables:       map[string]string{"requestID": "stale-request-id"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := api.CLIData{Steps: []api.CLIStep{{HTTPRequest: &tt.request}}}
			results := []api.CLIStepResult{{HTTPRequestResult: &tt.result}}

			event := LocalSubmissionEvent(data, results)
			if event.ResultSlug != api.VerificationResultSlugFailure {
				t.Fatalf("ResultSlug = %q, want failure", event.ResultSlug)
			}
			if event.StructuredErrCLI == nil {
				t.Fatal("expected structured failure")
			}
			if event.StructuredErrCLI.FailedStepIndex != 0 || event.StructuredErrCLI.FailedTestIndex != 1 {
				t.Fatalf("failure = %#v, want step 0 capture test 1", event.StructuredErrCLI)
			}
		})
	}
}

func TestLocalSubmissionEventAcceptsEmptyHTTPResponseCapture(t *testing.T) {
	request := api.CLIStepHTTPRequest{
		Tests: []api.HTTPRequestTest{{StatusCode: intPtr(200)}},
		ResponseVariables: []api.HTTPRequestResponseVariable{{
			Name:      "nextPath",
			BodyRegex: `next="([^"]*)"`,
		}},
	}
	data := api.CLIData{Steps: []api.CLIStep{{HTTPRequest: &request}}}
	results := []api.CLIStepResult{{HTTPRequestResult: &api.HTTPRequestResult{
		StatusCode: 200,
		BodyString: `next=""`,
		Variables:  map[string]string{"nextPath": ""},
	}}}

	event := LocalSubmissionEvent(data, results)
	if event.ResultSlug != api.VerificationResultSlugSuccess {
		t.Fatalf("ResultSlug = %q, want success; failure = %#v", event.ResultSlug, event.StructuredErrCLI)
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
