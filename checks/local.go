package checks

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/goccy/go-json"
)

func LocalSubmissionEvent(cliData api.CLIData, results []api.CLIStepResult) api.LessonSubmissionEvent {
	failure := EvaluateCLIResults(cliData, results)
	slug := api.VerificationResultSlugSuccess
	if failure != nil {
		slug = api.VerificationResultSlugFailure
		if failure.FailedStepIndex >= 0 &&
			failure.FailedStepIndex < len(cliData.Steps) &&
			cliData.Steps[failure.FailedStepIndex].NoPenaltyOnFail {
			slug = api.VerificationResultSlugNoop
		}
	}

	return api.LessonSubmissionEvent{
		ResultSlug:       slug,
		StructuredErrCLI: failure,
		XPReward:         -1,
	}
}

func EvaluateCLIResults(cliData api.CLIData, results []api.CLIStepResult) *api.StructuredErrCLI {
	for stepIndex, step := range cliData.Steps {
		if stepIndex >= len(results) {
			return localFailure(stepIndex, 0, "missing result for step")
		}

		switch {
		case step.CLICommand != nil:
			result := results[stepIndex].CLICommandResult
			if result == nil {
				return localFailure(stepIndex, 0, "missing CLI command result")
			}
			if failure := evaluateCLICommandTests(stepIndex, *step.CLICommand, *result); failure != nil {
				return failure
			}
		case step.HTTPRequest != nil:
			result := results[stepIndex].HTTPRequestResult
			if result == nil {
				return localFailure(stepIndex, 0, "missing HTTP request result")
			}
			if failure := evaluateHTTPRequestTests(stepIndex, *step.HTTPRequest, *result); failure != nil {
				return failure
			}
		default:
			return localFailure(stepIndex, 0, "missing step definition")
		}
	}

	return nil
}

func evaluateCLICommandTests(stepIndex int, cmd api.CLIStepCLICommand, result api.CLICommandResult) *api.StructuredErrCLI {
	if result.Err != "" {
		return localFailure(stepIndex, 0, result.Err)
	}

	for testIndex, test := range cmd.Tests {
		var err error

		switch {
		case test.ExitCode != nil:
			if result.ExitCode != *test.ExitCode {
				err = fmt.Errorf("expected exit code %d, got %d", *test.ExitCode, result.ExitCode)
			}
		case len(test.StdoutContainsAll) > 0:
			for _, contains := range test.StdoutContainsAll {
				needle := InterpolateVariables(contains, result.Variables)
				if !strings.Contains(result.Stdout, needle) {
					err = fmt.Errorf("expected stdout to contain %q", needle)
					break
				}
			}
		case len(test.StdoutContainsNone) > 0:
			for _, containsNone := range test.StdoutContainsNone {
				needle := InterpolateVariables(containsNone, result.Variables)
				if strings.Contains(result.Stdout, needle) {
					err = fmt.Errorf("expected stdout to not contain %q", needle)
					break
				}
			}
		case test.StdoutLinesGT != nil:
			lineCount := stdoutLineCount(result.Stdout)
			if lineCount <= *test.StdoutLinesGT {
				err = fmt.Errorf("expected stdout to have more than %d lines, got %d", *test.StdoutLinesGT, lineCount)
			}
		case test.StdoutJq != nil:
			err = evaluateStdoutJq(result.Stdout, *test.StdoutJq, result.Variables)
		default:
			err = fmt.Errorf("unsupported CLI command test")
		}

		if err != nil {
			return localFailure(stepIndex, testIndex, err.Error())
		}
	}

	return nil
}

func evaluateHTTPRequestTests(stepIndex int, req api.CLIStepHTTPRequest, result api.HTTPRequestResult) *api.StructuredErrCLI {
	if result.Err != "" {
		return localFailure(stepIndex, 0, result.Err)
	}

	for testIndex, test := range req.Tests {
		var err error

		switch {
		case test.StatusCode != nil:
			if result.StatusCode != *test.StatusCode {
				err = fmt.Errorf("expected status code %d, got %d", *test.StatusCode, result.StatusCode)
			}
		case test.BodyContains != nil:
			needle := InterpolateVariables(*test.BodyContains, result.Variables)
			if !strings.Contains(result.BodyString, needle) {
				err = fmt.Errorf("expected response body to contain %q", needle)
			}
		case test.BodyContainsNone != nil:
			needle := InterpolateVariables(*test.BodyContainsNone, result.Variables)
			if strings.Contains(result.BodyString, needle) {
				err = fmt.Errorf("expected response body to not contain %q", needle)
			}
		case test.HeadersEqual != nil:
			err = evaluateHeaderEquals(result.ResponseHeaders, *test.HeadersEqual, result.Variables, "header")
		case test.HeadersContain != nil:
			err = evaluateHeaderContains(result.ResponseHeaders, *test.HeadersContain, result.Variables, "header")
		case test.TrailersEqual != nil:
			err = evaluateHeaderEquals(result.ResponseTrailers, *test.TrailersEqual, result.Variables, "trailer")
		case test.TrailersContain != nil:
			err = evaluateHeaderContains(result.ResponseTrailers, *test.TrailersContain, result.Variables, "trailer")
		case test.JSONValue != nil:
			err = evaluateHTTPJSONValue(result.BodyString, *test.JSONValue, result.Variables)
		default:
			err = fmt.Errorf("unsupported HTTP request test")
		}

		if err != nil {
			return localFailure(stepIndex, testIndex, err.Error())
		}
	}

	captureIndex := len(req.Tests)
	for _, vardef := range req.ResponseVariables {
		expected := map[string]string{}
		if err := parseVariables([]byte(result.BodyString), []api.HTTPRequestResponseVariable{vardef}, expected); err != nil {
			return localFailure(stepIndex, captureIndex, err.Error())
		}

		want, found := expected[vardef.Name]
		if !found {
			return localFailure(stepIndex, captureIndex, fmt.Sprintf("missing value for response variable %q", vardef.Name))
		}
		got, captured := result.Variables[vardef.Name]
		if !captured || got != want {
			return localFailure(stepIndex, captureIndex, fmt.Sprintf("captured response variable %q did not match the response body", vardef.Name))
		}
	}

	if len(req.ResponseVariables) > 0 {
		captureIndex++
	}
	for _, vardef := range req.ResponseHeaderVariables {
		expected := map[string]string{}
		if err := parseHeaderVariables(result.ResponseHeaders, []api.HTTPRequestResponseHeaderVariable{vardef}, expected); err != nil {
			return localFailure(stepIndex, captureIndex, err.Error())
		}

		want, found := expected[vardef.Name]
		if !found {
			return localFailure(stepIndex, captureIndex, fmt.Sprintf("missing value for response header variable %q", vardef.Name))
		}
		got, captured := result.Variables[vardef.Name]
		if !captured || got != want {
			return localFailure(stepIndex, captureIndex, fmt.Sprintf("captured response header variable %q did not match the response header", vardef.Name))
		}
	}

	return nil
}

func evaluateHeaderEquals(headers map[string]string, test api.HTTPRequestTestHeader, variables map[string]string, label string) error {
	key := InterpolateVariables(test.Key, variables)
	want := InterpolateVariables(test.Value, variables)

	got, ok := findHeaderValue(headers, key)
	if !ok {
		return fmt.Errorf("expected %s %q to exist", label, key)
	}
	if got != want {
		return fmt.Errorf("expected %s %q to equal %q, got %q", label, key, want, got)
	}

	return nil
}

func evaluateHeaderContains(headers map[string]string, test api.HTTPRequestTestHeader, variables map[string]string, label string) error {
	key := InterpolateVariables(test.Key, variables)
	want := InterpolateVariables(test.Value, variables)

	got, ok := findHeaderValue(headers, key)
	if !ok {
		return fmt.Errorf("expected %s %q to exist", label, key)
	}
	if !strings.Contains(strings.ToLower(got), strings.ToLower(want)) {
		return fmt.Errorf("expected %s %q to contain %q, got %q", label, key, want, got)
	}

	return nil
}

func evaluateHTTPJSONValue(body string, test api.HTTPRequestTestJSONValue, variables map[string]string) error {
	got, err := valFromJqPath(test.Path, body)
	if err != nil {
		return err
	}

	want, err := httpJSONExpectedValue(test, variables)
	if err != nil {
		return err
	}

	if !compareValues(got, test.Operator, want) {
		return fmt.Errorf("expected JSON at %s %s %v, got %v", test.Path, test.Operator, want, got)
	}

	return nil
}

func httpJSONExpectedValue(test api.HTTPRequestTestJSONValue, variables map[string]string) (any, error) {
	switch {
	case test.IntValue != nil:
		return *test.IntValue, nil
	case test.StringValue != nil:
		return InterpolateVariables(*test.StringValue, variables), nil
	case test.BoolValue != nil:
		return *test.BoolValue, nil
	default:
		return nil, fmt.Errorf("missing expected JSON value")
	}
}

func evaluateStdoutJq(stdout string, test api.StdoutJqTest, variables map[string]string) error {
	queryText := InterpolateVariables(test.Query, variables)

	input, err := parseJqInput(stdout, test.InputMode)
	if err != nil {
		return err
	}

	results, err := executeJqQuery(queryText, input)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("jq query returned no results")
	}

outer:
	for _, expected := range test.ExpectedResults {
		if value, ok := expected.Value.(string); ok {
			expected.Value = InterpolateVariables(value, variables)
		}
		for _, actual := range results {
			if jqResultMatches(actual, expected) {
				continue outer
			}
		}
		return fmt.Errorf("expected jq results to contain %v", expected)
	}

	return nil
}

func jqResultMatches(actual any, expected api.JqExpectedResult) bool {
	switch expected.Type {
	case api.JqTypeString:
		got, gotOK := actual.(string)
		want, wantOK := expected.Value.(string)
		return gotOK && wantOK && expected.Operator == "==" && got == want
	case api.JqTypeBool:
		got, gotOK := coerceJqBool(actual)
		want, wantOK := coerceJqBool(expected.Value)
		return gotOK && wantOK && expected.Operator == "==" && got == want
	case api.JqTypeInt:
		got, gotOK := coerceJqInt(actual)
		want, wantOK := coerceJqInt(expected.Value)
		if !gotOK || !wantOK {
			return false
		}
		switch expected.Operator {
		case "==":
			return got == want
		case ">":
			return got > want
		case ">=":
			return got >= want
		case "<":
			return got < want
		case "<=":
			return got <= want
		}
	}
	return false
}

func coerceJqBool(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		parsed, err := strconv.ParseBool(v)
		return parsed, err == nil
	default:
		return false, false
	}
}

func coerceJqInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		if v < math.MinInt || v > math.MaxInt {
			return 0, false
		}
		return int(v), true
	case float64:
		// MaxInt rounds up as float64 on 64-bit hosts; use an exclusive upper bound.
		if math.IsNaN(v) || math.Trunc(v) != v || v < float64(math.MinInt) || v >= -float64(math.MinInt) {
			return 0, false
		}
		return int(v), true
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return coerceJqInt(parsed)
	case string:
		parsed, err := strconv.Atoi(v)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func compareValues(got any, operator api.OperatorType, want any) bool {
	switch operator {
	case api.OpEquals, "==":
		return valuesEqual(got, want)
	case api.OpGreaterThan, ">", ">=", "<", "<=":
		gotNum, gotOK := numberValue(got)
		wantNum, wantOK := numberValue(want)
		if !gotOK || !wantOK {
			return false
		}
		switch operator {
		case api.OpGreaterThan, ">":
			return gotNum > wantNum
		case ">=":
			return gotNum >= wantNum
		case "<":
			return gotNum < wantNum
		case "<=":
			return gotNum <= wantNum
		}
	case api.OpContains:
		return strings.Contains(fmt.Sprintf("%v", got), fmt.Sprintf("%v", want))
	case api.OpNotContains:
		return !strings.Contains(fmt.Sprintf("%v", got), fmt.Sprintf("%v", want))
	default:
		return false
	}
	return false
}

func valuesEqual(got any, want any) bool {
	if gotNum, gotOK := numberValue(got); gotOK {
		wantNum, wantOK := numberValue(want)
		return wantOK && math.Abs(gotNum-wantNum) < 0.000000001
	}
	return reflect.DeepEqual(got, want)
}

func numberValue(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case jsonNumber:
		parsed, err := strconv.ParseFloat(v.String(), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func stdoutLineCount(stdout string) int {
	if stdout == "" {
		return 0
	}
	return strings.Count(stdout, "\n") + 1
}

func localFailure(stepIndex int, testIndex int, message string) *api.StructuredErrCLI {
	return &api.StructuredErrCLI{
		ErrorMessage:    message,
		FailedStepIndex: stepIndex,
		FailedTestIndex: testIndex,
	}
}

type jsonNumber interface {
	String() string
}
