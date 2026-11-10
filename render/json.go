package render

import (
	"fmt"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/goccy/go-json"
)

func renderJSONMismatch(test api.HTTPRequestTestJSONValue, result api.HTTPRequestResult) string {
	var expected any
	switch {
	case test.BoolValue != nil:
		expected = *test.BoolValue
	case test.IntValue != nil:
		expected = *test.IntValue
	case test.StringValue != nil:
		expected = checks.InterpolateVariables(*test.StringValue, result.Variables)
	default:
		return ""
	}
	operator := ""
	switch test.Operator {
	case api.OpEquals:
	case api.OpGreaterThan:
		operator = "> "
	case api.OpContains:
		operator = "contains "
	case api.OpNotContains:
		operator = "does not contain "
	default:
		return ""
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return ""
	}
	actual, err := checks.JSONValueAtPath(test.Path, result.BodyString)
	actualText := ""
	if err != nil {
		actualText = fmt.Sprintf("unavailable (%s)", err)
	} else {
		actualJSON, err := json.Marshal(actual)
		if err != nil {
			return ""
		}
		actualText = string(actualJSON)
	}
	return fmt.Sprintf("JSON %s\n  Expected: %s%s\n  Actual:   %s", test.Path, operator, expectedJSON, actualText)
}
