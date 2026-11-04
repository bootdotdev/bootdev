package render

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/goccy/go-json"
)

func renderJSONMismatch(test api.HTTPRequestTestJSONValue, result api.HTTPRequestResult, evaluated bool, title string) string {
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
	expectedJSON, err := formatJSONValue(expected)
	if err != nil {
		return ""
	}
	expectedText := truncateVisualOutput(operator + expectedJSON)
	if !evaluated {
		return renderCheckValues(title, expectedText, "", false)
	}
	actual, err := checks.JSONValueAtPath(test.Path, result.BodyString)
	actualText := ""
	if err != nil {
		actualText = fmt.Sprintf("unavailable (%s)", err)
	} else {
		actualJSON, err := formatJSONValue(actual)
		if err != nil {
			return ""
		}
		actualText = actualJSON
	}
	return renderCheckValues(title, expectedText, truncateVisualOutput(actualText), true)
}

func formatJSONValue(value any) (string, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buffer.String(), "\n"), nil
}
