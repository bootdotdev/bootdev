package checks

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	api "github.com/bootdotdev/bootdev/client"
)

// Local grading mirrors the backend; success is represented by nil.
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
	if len(cliData.Steps) != len(results) {
		return localFailure(-1, -1, "wrong number of steps")
	}

	for i, step := range cliData.Steps {
		actual := results[i]

		if step.CLICommand != nil && actual.CLICommandResult != nil {
			verificationErr := evaluateCLICommandTests(i, *step.CLICommand, *actual.CLICommandResult)
			if verificationErr != nil {
				return verificationErr
			}
		} else if step.HTTPRequest != nil && actual.HTTPRequestResult != nil {
			verificationErr := evaluateHTTPRequestTests(i, *step.HTTPRequest, *actual.HTTPRequestResult)
			if verificationErr != nil {
				return verificationErr
			}
		} else {
			return localFailure(-1, -1, "invalid step")
		}
	}

	return nil
}

func evaluateCLICommandTests(stepIndex int, expect api.CLIStepCLICommand, actual api.CLICommandResult) *api.StructuredErrCLI {
	if err := validateCommandAssertions(expect); err != nil {
		return &api.StructuredErrCLI{ErrorMessage: err.Error(), FailedStepIndex: stepIndex, FailedTestIndex: -1}
	}
	if actual.ExitCode < 0 {
		return localFailure(stepIndex, -1, "failed to start command")
	}

	for i, expectedTest := range expect.Tests {
		if expectedTest.ExitCode != nil {
			if *expectedTest.ExitCode != actual.ExitCode {
				return localFailure(stepIndex, i, fmt.Sprintf("expected status code %v, got %v", *expectedTest.ExitCode, actual.ExitCode))
			}
		}
		if expectedTest.StdoutJq != nil {
			jqInput, err := parseJqInput(actual.Stdout, expectedTest.StdoutJq.InputMode)
			if err != nil {
				return localFailure(stepIndex, i, fmt.Sprintf("failed to read jq input: %v", err))
			}
			jqResults, err := executeJqQuery(expectedTest.StdoutJq.Query, jqInput)
			if err != nil {
				return localFailure(stepIndex, i, fmt.Sprintf("failed to run jq query: %v", err))
			}
			if len(jqResults) == 0 {
				return localFailure(stepIndex, i, "jq query returned no results")
			}
		outer:
			for _, expectedResult := range expectedTest.StdoutJq.ExpectedResults {
				for _, actualResult := range jqResults {
					if jqResultMatches(actualResult, expectedResult) {
						continue outer
					}
				}
				return localFailure(stepIndex, i, fmt.Sprintf("expected jq results to contain %v", expectedResult))
			}
		}
		if expectedTest.StdoutLinesGT != nil {
			count := strings.Count(actual.Stdout, "\n")
			if actual.Stdout != "" {
				count++
			}
			if count <= *expectedTest.StdoutLinesGT {
				return localFailure(stepIndex, i, fmt.Sprintf("expected more than %v lines, got %v", *expectedTest.StdoutLinesGT, count))
			}
		}
		if expectedTest.StdoutContainsAll != nil {
			for _, expectedContains := range expectedTest.StdoutContainsAll {
				interpolatedContains := InterpolateVariables(expectedContains, actual.Variables)
				if !strings.Contains(actual.Stdout, interpolatedContains) {
					return localFailure(stepIndex, i, fmt.Sprintf("expected stdout to contain %v", interpolatedContains))
				}
			}
		}
		if expectedTest.StdoutContainsNone != nil {
			for _, expectedContainsNone := range expectedTest.StdoutContainsNone {
				interpolatedContainsNone := InterpolateVariables(expectedContainsNone, actual.Variables)
				if strings.Contains(actual.Stdout, interpolatedContainsNone) {
					return localFailure(stepIndex, i, fmt.Sprintf("expected stdout to not contain %v", interpolatedContainsNone))
				}
			}
		}
	}

	return nil
}

func evaluateHTTPRequestTests(stepIndex int, expect api.CLIStepHTTPRequest, actual api.HTTPRequestResult) *api.StructuredErrCLI {
	if err := validateHTTPAssertions(expect); err != nil {
		return &api.StructuredErrCLI{ErrorMessage: err.Error(), FailedStepIndex: stepIndex, FailedTestIndex: -1}
	}
	if actual.Err != "" {
		return localFailure(stepIndex, -1, fmt.Sprintf("fetch error: %v", actual.Err))
	}

	for i, expectedTest := range expect.Tests {
		if expectedTest.StatusCode != nil {
			if *expectedTest.StatusCode != actual.StatusCode {
				return localFailure(stepIndex, i, fmt.Sprintf("expected status code %v, got %v", *expectedTest.StatusCode, actual.StatusCode))
			}
		}

		if expectedTest.BodyContains != nil {
			if !strings.Contains(actual.BodyString, *expectedTest.BodyContains) {
				return localFailure(stepIndex, i, fmt.Sprintf("expected response body to contain '%v', but it did not", *expectedTest.BodyContains))
			}
		}

		if expectedTest.BodyContainsNone != nil {
			if strings.Contains(actual.BodyString, *expectedTest.BodyContainsNone) {
				return localFailure(stepIndex, i, fmt.Sprintf("expected response body to not contain '%v', but it did", *expectedTest.BodyContainsNone))
			}
		}

		if expectedTest.HeadersEqual != nil {
			actualHeaderValue, ok := findHeaderValue(actual.ResponseHeaders, expectedTest.HeadersEqual.Key)
			if !ok || actualHeaderValue != expectedTest.HeadersEqual.Value {
				return localFailure(stepIndex, i, fmt.Sprintf("expected '%v' header to equal '%v', but it did not", expectedTest.HeadersEqual.Key, expectedTest.HeadersEqual.Value))
			}
		}

		if expectedTest.HeadersContain != nil {
			actualHeaderValue, ok := findHeaderValue(actual.ResponseHeaders, expectedTest.HeadersContain.Key)
			if !ok || !strings.Contains(actualHeaderValue, expectedTest.HeadersContain.Value) {
				return localFailure(stepIndex, i, fmt.Sprintf("expected '%v' header to contain '%v', but it did not", expectedTest.HeadersContain.Key, expectedTest.HeadersContain.Value))
			}
		}

		if expectedTest.TrailersEqual != nil {
			actualTrailerValue, ok := findHeaderValue(actual.ResponseTrailers, expectedTest.TrailersEqual.Key)
			if !ok || actualTrailerValue != expectedTest.TrailersEqual.Value {
				return localFailure(stepIndex, i, fmt.Sprintf("expected '%v' trailer to equal '%v', but it did not", expectedTest.TrailersEqual.Key, expectedTest.TrailersEqual.Value))
			}
		}

		if expectedTest.TrailersContain != nil {
			actualTrailerValue, ok := findHeaderValue(actual.ResponseTrailers, expectedTest.TrailersContain.Key)
			if !ok || !strings.Contains(actualTrailerValue, expectedTest.TrailersContain.Value) {
				return localFailure(stepIndex, i, fmt.Sprintf("expected '%v' trailer to contain '%v', but it did not", expectedTest.TrailersContain.Key, expectedTest.TrailersContain.Value))
			}
		}

		if expectedTest.JSONValue != nil {
			err := jsonValOp(*expectedTest.JSONValue, actual.BodyString, actual.Variables)
			if err != nil {
				return localFailure(stepIndex, i, fmt.Sprintf("%v", err))
			}
		}
	}

	responseVariableTestIndex := len(expect.Tests) + 1
	responseHeaderVariableTestIndex := responseVariableTestIndex
	if len(expect.ResponseVariables) > 0 {
		responseHeaderVariableTestIndex++
	}

	for _, expectedVar := range expect.ResponseVariables {
		expectedValue, ok := responseVariableValue(expectedVar, actual.BodyString)
		if !ok {
			return localFailure(stepIndex, responseVariableTestIndex, fmt.Sprintf("missing value for variable '%s'", expectedVar.Name))
		}

		if !capturedVariableMatches(actual.Variables, expectedVar.Name, expectedValue) {
			return localFailure(stepIndex, responseVariableTestIndex, fmt.Sprintf("captured variable '%s' did not match expected response body value", expectedVar.Name))
		}
	}

	for _, expectedVar := range expect.ResponseHeaderVariables {
		expectedValue, ok := responseHeaderVariableValue(expectedVar, actual.ResponseHeaders)
		if !ok {
			return localFailure(stepIndex, responseHeaderVariableTestIndex, fmt.Sprintf("missing value for variable '%s'", expectedVar.Name))
		}

		if !capturedVariableMatches(actual.Variables, expectedVar.Name, expectedValue) {
			return localFailure(stepIndex, responseHeaderVariableTestIndex, fmt.Sprintf("captured variable '%s' did not match expected response header value", expectedVar.Name))
		}
	}

	return nil
}

func capturedVariableMatches(vars map[string]string, name, expectedValue string) bool {
	actualValue, ok := vars[name]
	return ok && actualValue == expectedValue
}

func responseVariableValue(expectedVar api.HTTPRequestResponseVariable, body string) (string, bool) {
	if expectedVar.Path != "" {
		val, err := valFromJqPath(expectedVar.Path, body)
		if err != nil || val == nil {
			return "", false
		}
		return fmt.Sprintf("%v", val), true
	}

	re, err := regexp.Compile(expectedVar.BodyRegex)
	if err != nil {
		return "", false
	}

	matches := re.FindStringSubmatch(body)
	if len(matches) != 2 {
		return "", false
	}

	return matches[1], true
}

func responseHeaderVariableValue(expectedVar api.HTTPRequestResponseHeaderVariable, headers map[string]string) (string, bool) {
	headerValue, ok := findHeaderValue(headers, expectedVar.Header)
	if !ok {
		return "", false
	}

	if expectedVar.Regex == "" {
		return headerValue, true
	}

	re, err := regexp.Compile(expectedVar.Regex)
	if err != nil {
		return "", false
	}

	matches := re.FindStringSubmatch(headerValue)
	if len(matches) != 2 {
		return "", false
	}

	return matches[1], true
}

func jsonValOp(test api.HTTPRequestTestJSONValue, jsn string, variables map[string]string) error {
	val, err := valFromJqPath(test.Path, jsn)
	if err != nil {
		return err
	}
	if test.BoolValue != nil {
		vBool, ok := val.(bool)
		if !ok {
			return errors.New("expected boolean value")
		}
		if test.Operator == api.OpEquals {
			if vBool != *test.BoolValue {
				return errors.New("boolean value not equal")
			}
			return nil
		}
		return errors.New("operator not supported")
	}
	if test.IntValue != nil {
		var v int
		vInt, intOk := val.(int)
		vFloat, floatOk := val.(float64)
		switch {
		case intOk:
			v = vInt
		case floatOk:
			v = int(vFloat)
		default:
			return errors.New("expected int value")
		}
		if test.Operator == api.OpEquals {
			if v != *test.IntValue {
				return errors.New("int value not equal")
			}
			return nil
		}
		if test.Operator == api.OpGreaterThan {
			if v <= *test.IntValue {
				return errors.New("int value not greater than")
			}
			return nil
		}
		return errors.New("operator not supported")
	}
	if test.StringValue != nil {
		vStr, ok := val.(string)
		if !ok {
			return errors.New("expected string value")
		}
		if test.Operator == api.OpEquals {
			interpolatedStr := InterpolateVariables(*test.StringValue, variables)
			if vStr != interpolatedStr {
				return errors.New("string value not equal")
			}
			return nil
		}
		if test.Operator == api.OpContains {
			interpolatedStr := InterpolateVariables(*test.StringValue, variables)
			if !strings.Contains(vStr, interpolatedStr) {
				return fmt.Errorf("%s does not contain %s", vStr, interpolatedStr)
			}
			return nil
		}
		if test.Operator == api.OpNotContains {
			interpolatedStr := InterpolateVariables(*test.StringValue, variables)
			if strings.Contains(vStr, interpolatedStr) {
				return fmt.Errorf("%s contains %s", vStr, interpolatedStr)
			}
			return nil
		}
		return errors.New("operator not supported")
	}

	return errors.New("no test value provided")
}

func jqResultMatches(actualResult any, expectedResult api.JqExpectedResult) bool {
	switch expectedResult.Type {
	case api.JqTypeBool:
		expected, expectedOk := coerceBool(expectedResult.Value)
		actual, actualOk := coerceBool(actualResult)
		if !expectedOk || !actualOk {
			return false
		}
		return compareBool(actual, expected, expectedResult.Operator)
	case api.JqTypeString:
		expected, expectedOk := coerceString(expectedResult.Value)
		actual, actualOk := coerceString(actualResult)
		if !expectedOk || !actualOk {
			return false
		}
		return compareString(actual, expected, expectedResult.Operator)
	case api.JqTypeInt:
		expected, expectedOk := coerceInt(expectedResult.Value)
		actual, actualOk := coerceInt(actualResult)
		if !expectedOk || !actualOk {
			return false
		}
		return compareInt(actual, expected, expectedResult.Operator)
	default:
		return false
	}
}

func coerceBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(typed)
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func coerceString(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	default:
		return "", false
	}
}

func coerceInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		if typed > math.MaxInt || typed < math.MinInt {
			return 0, false
		}
		return int(typed), true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return 0, false
		}
		if math.Trunc(typed) != typed {
			return 0, false
		}
		// MaxInt rounds up as float64 on 64-bit hosts; use an exclusive upper bound.
		if typed >= -float64(math.MinInt) || typed < float64(math.MinInt) {
			return 0, false
		}
		return int(typed), true
	case json.Number:
		parsed, ok := new(big.Rat).SetString(typed.String())
		if !ok || !parsed.IsInt() || !parsed.Num().IsInt64() {
			return 0, false
		}
		return coerceInt(parsed.Num().Int64())
	case string:
		parsed, err := strconv.Atoi(typed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func compareBool(actual bool, expected bool, operator api.JqOperator) bool {
	switch operator {
	case "==":
		return actual == expected
	default:
		return false
	}
}

func compareString(actual string, expected string, operator api.JqOperator) bool {
	switch operator {
	case "==":
		return actual == expected
	default:
		return false
	}
}

func compareInt(actual int, expected int, operator api.JqOperator) bool {
	switch operator {
	case "==":
		return actual == expected
	case ">":
		return actual > expected
	case ">=":
		return actual >= expected
	case "<":
		return actual < expected
	case "<=":
		return actual <= expected
	default:
		return false
	}
}

func localFailure(stepIndex, testIndex int, message string) *api.StructuredErrCLI {
	return &api.StructuredErrCLI{ErrorMessage: message, FailedStepIndex: stepIndex, FailedTestIndex: testIndex}
}
