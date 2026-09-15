package checks

import (
	"errors"
	"fmt"
	"regexp"

	api "github.com/bootdotdev/bootdev/client"
)

func validateCLIAssertions(cliData api.CLIData) error {
	for stepIndex, step := range cliData.Steps {
		if (step.CLICommand == nil) == (step.HTTPRequest == nil) {
			return fmt.Errorf("step %d must contain exactly one command or HTTP request", stepIndex+1)
		}
		var err error
		if step.CLICommand != nil {
			err = validateCommandAssertions(*step.CLICommand)
		} else {
			err = validateHTTPAssertions(*step.HTTPRequest)
		}
		if err != nil {
			return fmt.Errorf("step %d: %w", stepIndex+1, err)
		}
	}
	return nil
}

func validateCommandAssertions(command api.CLIStepCLICommand) error {
	for testIndex, test := range command.Tests {
		if test.ExitCode == nil && test.StdoutLinesGT == nil && test.StdoutJq == nil && len(test.StdoutContainsAll) == 0 && len(test.StdoutContainsNone) == 0 {
			return fmt.Errorf("test %d contains no assertions", testIndex+1)
		}
		if test.StdoutJq == nil {
			continue
		}
		if test.StdoutJq.Query == "" || len(test.StdoutJq.ExpectedResults) == 0 {
			return fmt.Errorf("test %d requires a jq query and expected results", testIndex+1)
		}
		for _, expected := range test.StdoutJq.ExpectedResults {
			if expected.Type != api.JqTypeBool && expected.Type != api.JqTypeString && expected.Type != api.JqTypeInt {
				return errors.New("invalid jq expected type")
			}
			switch expected.Operator {
			case "==", ">", ">=", "<", "<=":
			default:
				return errors.New("invalid jq operator")
			}
			if expected.Value == nil {
				return errors.New("missing jq expected value")
			}
		}
	}
	return nil
}

func validateHTTPAssertions(request api.CLIStepHTTPRequest) error {
	for testIndex, test := range request.Tests {
		if test.StatusCode == nil && test.BodyContains == nil && test.BodyContainsNone == nil && test.HeadersEqual == nil && test.HeadersContain == nil && test.TrailersEqual == nil && test.TrailersContain == nil && test.JSONValue == nil {
			return fmt.Errorf("test %d contains no assertions", testIndex+1)
		}
		if test.JSONValue == nil {
			continue
		}
		valueCount := 0
		if test.JSONValue.IntValue != nil {
			valueCount++
		}
		if test.JSONValue.StringValue != nil {
			valueCount++
		}
		if test.JSONValue.BoolValue != nil {
			valueCount++
		}
		if valueCount != 1 {
			return errors.New("JSON assertion requires exactly one expected value type")
		}
		if test.JSONValue.Path == "" {
			return errors.New("JSON assertion requires a path")
		}
		switch test.JSONValue.Operator {
		case api.OpEquals, api.OpGreaterThan, api.OpContains, api.OpNotContains:
		default:
			return errors.New("invalid JSON assertion operator")
		}
	}
	for _, capture := range request.ResponseVariables {
		if (capture.Path == "") == (capture.BodyRegex == "") {
			return errors.New("response variable requires exactly one of path or bodyRegex")
		}
		if capture.BodyRegex != "" {
			if err := validateCaptureRegex(capture.BodyRegex); err != nil {
				return err
			}
		}
	}
	for _, capture := range request.ResponseHeaderVariables {
		if capture.Regex != "" {
			if err := validateCaptureRegex(capture.Regex); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateCaptureRegex(pattern string) error {
	expression, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	if expression.NumSubexp() != 1 {
		return errors.New("capture regex requires exactly one capture group")
	}
	return nil
}
