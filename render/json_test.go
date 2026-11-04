package render

import (
	"strings"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
)

func TestFailedJSONCheckShowsSelectedValue(t *testing.T) {
	expectedString := "Darn that fly, I just wanna cook"
	variableString := "${name}"
	expectedInt := 3
	expectedBool := true
	for _, tt := range []struct {
		name string
		test api.HTTPRequestTestJSONValue
		body string
		want string
	}{
		{name: "array field", test: api.HTTPRequestTestJSONValue{Path: ".[0].body", Operator: api.OpEquals, StringValue: &expectedString}, body: `[{"body":"I'm the one who knocks!"}]`, want: "JSON .[0].body\n  Expected: \"Darn that fly, I just wanna cook\"\n  Actual:   \"I'm the one who knocks!\""},
		{name: "interpolation", test: api.HTTPRequestTestJSONValue{Path: ".name", Operator: api.OpEquals, StringValue: &variableString}, body: `{"name":"other"}`, want: "Expected: \"Walt\"\n  Actual:   \"other\""},
		{name: "integer comparison", test: api.HTTPRequestTestJSONValue{Path: ".count", Operator: api.OpGreaterThan, IntValue: &expectedInt}, body: `{"count":2}`, want: "Expected: > 3\n  Actual:   2"},
		{name: "boolean", test: api.HTTPRequestTestJSONValue{Path: ".ok", Operator: api.OpEquals, BoolValue: &expectedBool}, body: `{"ok":false}`, want: "Expected: true\n  Actual:   false"},
		{name: "type mismatch", test: api.HTTPRequestTestJSONValue{Path: ".count", Operator: api.OpEquals, IntValue: &expectedInt}, body: `{"count":"3"}`, want: "Expected: 3\n  Actual:   \"3\""},
		{name: "contains", test: api.HTTPRequestTestJSONValue{Path: ".name", Operator: api.OpContains, StringValue: &variableString}, body: `{"name":"other"}`, want: "Expected: contains \"Walt\""},
		{name: "not contains", test: api.HTTPRequestTestJSONValue{Path: ".name", Operator: api.OpNotContains, StringValue: &variableString}, body: `{"name":"Walter"}`, want: "Expected: does not contain \"Walt\"\n  Actual:   \"Walter\""},
		{name: "missing", test: api.HTTPRequestTestJSONValue{Path: ".name", Operator: api.OpEquals, StringValue: &variableString}, body: `{}`, want: "Actual:   unavailable (value not found)"},
		{name: "multiple results", test: api.HTTPRequestTestJSONValue{Path: ".[].name", Operator: api.OpEquals, StringValue: &variableString}, body: `[{"name":"one"},{"name":"two"}]`, want: "Actual:   unavailable (invalid number of values found)"},
		{name: "invalid JSON", test: api.HTTPRequestTestJSONValue{Path: ".name", Operator: api.OpEquals, StringValue: &variableString}, body: `not json`, want: "Actual:   unavailable ("},
		{name: "invalid query", test: api.HTTPRequestTestJSONValue{Path: ".[", Operator: api.OpEquals, StringValue: &variableString}, body: `{}`, want: "Actual:   unavailable ("},
	} {
		t.Run(tt.name, func(t *testing.T) {
			step := stepModel{
				tests: []testModel{{text: "original condition"}},
				result: &api.CLIStepResult{HTTPRequestResult: &api.HTTPRequestResult{
					BodyString: tt.body, Variables: map[string]string{"name": "Walt"},
					Request: api.CLIStepHTTPRequest{Tests: []api.HTTPRequestTest{{JSONValue: &tt.test}}},
				}},
			}
			got := renderCheck(step, 0, "FAIL")
			if !strings.HasPrefix(got, "FAIL · JSON ") || !strings.Contains(got, tt.want) {
				t.Fatalf("expected %q in:\n%s", tt.want, got)
			}
			check := renderCheck(step, 0, "PASS")
			if strings.TrimPrefix(check, "PASS · ") != strings.TrimPrefix(got, "FAIL · ") {
				t.Fatalf("check details should not depend on grading status: %s", check)
			}
		})
	}
}

func TestCompoundJSONCheckRetainsOtherConditions(t *testing.T) {
	expectedStatus, expectedInt := 200, 3
	step := stepModel{
		tests: []testModel{{text: "status and JSON conditions"}},
		result: &api.CLIStepResult{HTTPRequestResult: &api.HTTPRequestResult{
			Request: api.CLIStepHTTPRequest{Tests: []api.HTTPRequestTest{{
				StatusCode: &expectedStatus,
				JSONValue:  &api.HTTPRequestTestJSONValue{Path: ".count", Operator: api.OpEquals, IntValue: &expectedInt},
			}}},
		}},
	}
	if got := renderCheck(step, 0, "FAIL"); got != "FAIL · status and JSON conditions" {
		t.Fatalf("compound assertion lost context: %s", got)
	}
}
