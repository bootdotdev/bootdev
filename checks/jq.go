package checks

import (
	"errors"
	"fmt"
	"io"
	"strings"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/goccy/go-json"
	"github.com/itchyny/gojq"
)

func prettyPrintStdoutJqTest(test api.StdoutJqTest, variables map[string]string) string {
	queryText := InterpolateVariables(test.Query, variables)
	var str strings.Builder
	str.WriteString(fmt.Sprintf("Expect jq query '%s' to yield values satisfying:", queryText))
	if len(test.ExpectedResults) == 0 {
		str.WriteString("\n       - [no expected results provided]")
		return str.String()
	}
	for _, expected := range test.ExpectedResults {
		value := formatJqExpectedValue(expected, variables)
		fmt.Fprintf(&str, "\n       - %s %s %s", expected.Type, expected.Operator, value)
	}
	return str.String()
}

func formatJqExpectedValue(expected api.JqExpectedResult, variables map[string]string) string {
	value := expected.Value
	if expected.Type == api.JqTypeString {
		if stringValue, ok := expected.Value.(string); ok {
			value = InterpolateVariables(stringValue, variables)
		}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(encoded)
}

func collectStdoutJqOutputs(cmd api.CLIStepCLICommand, result api.CLICommandResult) []api.CLICommandJqOutput {
	var outputs []api.CLICommandJqOutput
	for _, test := range cmd.Tests {
		if test.StdoutJq == nil {
			continue
		}
		outputs = append(outputs, runStdoutJqQuery(result.Stdout, *test.StdoutJq, result.Variables))
	}
	return outputs
}

func runStdoutJqQuery(stdout string, test api.StdoutJqTest, variables map[string]string) api.CLICommandJqOutput {
	queryText := InterpolateVariables(test.Query, variables)
	input, err := parseJqInput(stdout, test.InputMode)
	if err != nil {
		return api.CLICommandJqOutput{Query: queryText, Error: err.Error()}
	}
	results, err := executeJqQuery(queryText, input)
	if err != nil {
		return api.CLICommandJqOutput{Query: queryText, Error: err.Error()}
	}
	return api.CLICommandJqOutput{Query: queryText, Results: formatJqResults(results)}
}

func parseJqInput(stdout string, inputMode string) (any, error) {
	mode := strings.ToLower(strings.TrimSpace(inputMode))
	if mode != "json" && mode != "jsonl" {
		mode = "json"
	}

	decoder := json.NewDecoder(strings.NewReader(stdout))
	decoder.UseNumber()
	if mode == "jsonl" {
		var values []any
		for {
			var value any
			err := decoder.Decode(&value)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, nil
	}

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("expected a single JSON value")
		}
		return nil, err
	}

	return value, nil
}

func executeJqQuery(queryText string, input any) ([]any, error) {
	query, err := gojq.Parse(queryText)
	if err != nil {
		return nil, err
	}
	iter := query.Run(input)
	var results []any
	for {
		val, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := val.(error); ok {
			return nil, err
		}
		results = append(results, val)
	}
	return results, nil
}

func formatJqResults(results []any) []string {
	if len(results) == 0 {
		return nil
	}
	formatted := make([]string, 0, len(results))
	for _, result := range results {
		if result == nil {
			formatted = append(formatted, "null")
			continue
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			formatted = append(formatted, fmt.Sprintf("%v", result))
			continue
		}
		formatted = append(formatted, string(encoded))
	}
	return formatted
}

func valFromJqPath(path string, jsn string) (any, error) {
	vals, err := valsFromJqPath(path, jsn)
	if err != nil {
		return nil, err
	}
	if len(vals) != 1 {
		return nil, errors.New("invalid number of values found")
	}
	val := vals[0]
	if val == nil {
		return nil, errors.New("value not found")
	}
	return val, nil
}

func valsFromJqPath(path string, jsn string) ([]any, error) {
	var parseable any
	err := json.Unmarshal([]byte(jsn), &parseable)
	if err != nil {
		return nil, err
	}

	query, err := gojq.Parse(path)
	if err != nil {
		return nil, err
	}
	iter := query.Run(parseable)
	vals := []any{}
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
				break
			}
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, nil
}
