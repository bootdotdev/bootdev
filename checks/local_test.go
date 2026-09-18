package checks

import (
	"math"
	"strconv"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/goccy/go-json"
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

func TestLocalSubmissionEventRejectsCommandCaptureError(t *testing.T) {
	command := api.CLIStepCLICommand{
		Command: "echo hello",
		StdoutVariables: []api.CLICommandStdoutVariable{{
			Name: "value", Regex: "(",
		}},
		Tests: []api.CLICommandTest{{ExitCode: intPtr(0)}},
	}
	result := runCLICommand(command, map[string]string{}, defaultShell())
	if result.ExitCode != 0 || result.Err == "" {
		t.Fatalf("expected successful command with capture error, got %#v", result)
	}
	event := LocalSubmissionEvent(
		api.CLIData{Steps: []api.CLIStep{{CLICommand: &command}}},
		[]api.CLIStepResult{{CLICommandResult: &result}},
	)
	if event.ResultSlug != api.VerificationResultSlugFailure || event.StructuredErrCLI == nil {
		t.Fatalf("expected capture error to fail grading, got %#v", event)
	}
	failure := event.StructuredErrCLI
	if failure.ErrorMessage != result.Err || failure.FailedStepIndex != 0 || failure.FailedTestIndex != -1 {
		t.Fatalf("unexpected failure: %#v", failure)
	}
}

func TestEvaluateStdoutJqNumericComparisons(t *testing.T) {
	for _, tt := range []struct {
		operator api.JqOperator
		pass     [3]bool
	}{
		{"==", [3]bool{false, true, false}},
		{">", [3]bool{false, false, true}},
		{">=", [3]bool{false, true, true}},
		{"<", [3]bool{true, false, false}},
		{"<=", [3]bool{true, true, false}},
	} {
		for i, stdout := range []string{"4", "5", "6"} {
			t.Run(stdout+string(tt.operator)+"5", func(t *testing.T) {
				jqTest := api.StdoutJqTest{
					InputMode: "json",
					Query:     ".",
					ExpectedResults: []api.JqExpectedResult{{
						Type:     api.JqTypeInt,
						Operator: tt.operator,
						Value:    5,
					}},
				}
				err := evaluateCLICommandTests(0, api.CLIStepCLICommand{Tests: []api.CLICommandTest{{StdoutJq: &jqTest}}}, api.CLICommandResult{Stdout: stdout})
				if (err == nil) != tt.pass[i] {
					t.Fatalf("comparison passed = %t, want %t; error: %v", err == nil, tt.pass[i], err)
				}
			})
		}
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
			if event.StructuredErrCLI.FailedStepIndex != 0 || event.StructuredErrCLI.FailedTestIndex != 2 {
				t.Fatalf("failure = %#v, want step 0 capture test 2", event.StructuredErrCLI)
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

func intPtr(v int) *int {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func TestEvaluateStdoutJqMatchesAnyResult(t *testing.T) {
	for _, tt := range []struct {
		name     string
		stdout   string
		expected []int
		pass     bool
	}{
		{"extra and reordered results", "[3, 2, 1]", []int{1, 2}, true},
		{"reuse an actual result", "[1]", []int{1, 1}, true},
		{"missing expected result", "[1, 3]", []int{1, 2}, false},
		{"empty results", "[]", []int{1}, false},
		{"empty results without expectations", "[]", nil, false},
		{"nonempty results without expectations", "[1]", nil, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			test := api.StdoutJqTest{InputMode: "json", Query: ".[]"}
			for _, value := range tt.expected {
				test.ExpectedResults = append(test.ExpectedResults, api.JqExpectedResult{
					Type: api.JqTypeInt, Operator: "==", Value: value,
				})
			}
			err := evaluateCLICommandTests(0, api.CLIStepCLICommand{Tests: []api.CLICommandTest{{StdoutJq: &test}}}, api.CLICommandResult{Stdout: tt.stdout})
			if (err == nil) != tt.pass {
				t.Fatalf("passed = %t, want %t; error: %v", err == nil, tt.pass, err)
			}
		})
	}
}

func TestEvaluateStdoutJqResultTypes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		stdout   string
		kind     api.JqValueType
		operator api.JqOperator
		want     any
		pass     bool
	}{
		{"numeric string", `"5"`, api.JqTypeInt, "==", 5, true},
		{"integer expectation is literal", "5", api.JqTypeInt, "==", "${value}", false},
		{"integral expected float", "5", api.JqTypeInt, "==", 5.0, true},
		{"decimal JSON number", "5.0", api.JqTypeInt, "==", 5, true},
		{"exponent JSON number", "5e0", api.JqTypeInt, "==", 5, true},
		{"tiny fractional part", "5.0000000000000000001", api.JqTypeInt, "==", 5, false},
		{"exact maximum integer", strconv.Itoa(math.MaxInt) + ".0", api.JqTypeInt, "==", math.MaxInt, true},
		{"JSON integer overflow", "9223372036854775808.0", api.JqTypeInt, ">", 0, false},
		{"fractional actual", "5.5", api.JqTypeInt, ">", 5, false},
		{"fractional expected", "6", api.JqTypeInt, ">", 5.5, false},
		{"exact large integer", "9007199254740992", api.JqTypeInt, "==", json.Number("9007199254740993"), false},
		{"out of range float", "0", api.JqTypeInt, "<=", -float64(math.MinInt), false},
		{"boolean", "true", api.JqTypeBool, "==", true, true},
		{"boolean strings", `"true"`, api.JqTypeBool, "==", "true", true},
		{"invalid boolean", `"yes"`, api.JqTypeBool, "==", true, false},
		{"string expectation is literal", `"5"`, api.JqTypeString, "==", "${value}", false},
		{"string type rejects numbers", "5", api.JqTypeString, "==", 5, false},
		{"boolean type rejects numbers", "1", api.JqTypeBool, "==", 1, false},
		{"string ordering unsupported", `"b"`, api.JqTypeString, ">", "a", false},
		{"unknown operator", "5", api.JqTypeInt, "!=", 4, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			jqTest := api.StdoutJqTest{
				InputMode: "json",
				Query:     ".",
				ExpectedResults: []api.JqExpectedResult{{
					Type: tt.kind, Operator: tt.operator, Value: tt.want,
				}},
			}
			err := evaluateCLICommandTests(0, api.CLIStepCLICommand{Tests: []api.CLICommandTest{{StdoutJq: &jqTest}}}, api.CLICommandResult{Stdout: tt.stdout, Variables: map[string]string{"value": "5"}})
			if (err == nil) != tt.pass {
				t.Fatalf("passed = %t, want %t; error: %v", err == nil, tt.pass, err)
			}
		})
	}
}
