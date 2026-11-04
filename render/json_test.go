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
	expectedHTML := "<b>Hello & goodbye</b>"
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
		{name: "HTML characters", test: api.HTTPRequestTestJSONValue{Path: ".html", Operator: api.OpEquals, StringValue: &expectedHTML}, body: `{"html":"<i>Hello & goodbye</i>"}`, want: "Expected: \"<b>Hello & goodbye</b>\"\n  Actual:   \"<i>Hello & goodbye</i>\""},
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
			passed, failed := true, false
			got, _ := renderCheckDetails(step, 0, &failed)
			if !strings.HasPrefix(got, "FAIL · 1. JSON ") || !strings.Contains(got, tt.want) {
				t.Fatalf("expected %q in:\n%s", tt.want, got)
			}
			check, _ := renderCheckDetails(step, 0, &passed)
			if strings.TrimPrefix(check, "PASS · ") != strings.TrimPrefix(got, "FAIL · ") {
				t.Fatalf("check details should not depend on grading status: %s", check)
			}
			unevaluated, _ := renderCheckDetails(step, 0, nil)
			if strings.Contains(unevaluated, "Actual:") || !strings.Contains(unevaluated, "Expected:") {
				t.Fatalf("unevaluated check should show only the expectation: %s", unevaluated)
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
	failed := false
	if got, _ := renderCheckDetails(step, 0, &failed); got != "FAIL · 1. status and JSON conditions" {
		t.Fatalf("compound assertion lost context: %s", got)
	}
}

func TestJSONCheckTruncatesLargeExpectedAndActualValues(t *testing.T) {
	expected := strings.Repeat("界", 20000)
	actual := strings.Repeat("x", 20000)
	test := api.HTTPRequestTestJSONValue{Path: ".value", Operator: api.OpContains, StringValue: &expected}
	result := api.HTTPRequestResult{BodyString: `{"value":"` + actual + `"}`}
	for _, evaluated := range []bool{false, true} {
		got := renderJSONMismatch(test, result, evaluated, "JSON .value")
		wantMarkers := 1
		if evaluated {
			wantMarkers = 2
		}
		if strings.Count(got, "... output visually truncated") != wantMarkers {
			t.Fatalf("evaluated=%v: expected %d truncation markers", evaluated, wantMarkers)
		}
		if strings.Contains(got, strings.Repeat("界", 5120)) || strings.Contains(got, strings.Repeat("x", 5120)) {
			t.Fatalf("evaluated=%v: JSON value exceeds visual limit", evaluated)
		}
		if strings.Contains(got, "Actual:") != evaluated {
			t.Fatalf("evaluated=%v: unexpected actual value visibility", evaluated)
		}
	}
}
