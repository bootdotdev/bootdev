package checks

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/goccy/go-json"
	"github.com/itchyny/gojq"
	"github.com/tailscale/hujson"
)

func prettyPrintStdoutJqTest(test api.StdoutJqTest) string {
	var str strings.Builder
	fmt.Fprintf(&str, "Expect jq query '%s' to yield values satisfying:", test.Query)
	if len(test.ExpectedResults) == 0 {
		str.WriteString("\n       - [no expected results provided]")
		return str.String()
	}
	for _, expected := range test.ExpectedResults {
		value := formatJqExpectedValue(expected)
		fmt.Fprintf(&str, "\n       - %s %s %s", expected.Type, expected.Operator, value)
	}
	return str.String()
}

func formatJqExpectedValue(expected api.JqExpectedResult) string {
	value := expected.Value
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
		outputs = append(outputs, runStdoutJqQuery(result.Stdout, *test.StdoutJq))
	}
	return outputs
}

func runStdoutJqQuery(stdout string, test api.StdoutJqTest) api.CLICommandJqOutput {
	input, err := parseJqInput(stdout, test.InputMode)
	if err != nil {
		return api.CLICommandJqOutput{Query: test.Query, Error: err.Error()}
	}
	results, err := executeJqQuery(test.Query, input)
	if err != nil {
		return api.CLICommandJqOutput{Query: test.Query, Error: err.Error()}
	}
	return api.CLICommandJqOutput{Query: test.Query, Results: formatJqResults(results)}
}

func parseJqInput(stdout string, inputMode string) (any, error) {
	mode := strings.ToLower(strings.TrimSpace(inputMode))
	var inputReader io.Reader
	if mode != "jsonl" {
		// HuJSON requires a newline to terminate a final line comment.
		standardJSON, err := hujson.Standardize([]byte(stdout + "\n"))
		if err != nil {
			return nil, err
		}
		inputReader = bytes.NewReader(standardJSON)
	} else {
		inputReader = strings.NewReader(stdout)
	}

	decoder := json.NewDecoder(inputReader)
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

func jqResultMatches(actualResult any, expectedResult api.JqExpectedResult) bool {
	switch expectedResult.Type {
	case api.JqTypeBool:
		expected, expectedOk := coerceBool(expectedResult.Value)
		actual, actualOk := coerceBool(actualResult)
		if !expectedOk || !actualOk {
			return false
		}
		return expectedResult.Operator == "==" && actual == expected
	case api.JqTypeString:
		expected, expectedOk := expectedResult.Value.(string)
		actual, actualOk := actualResult.(string)
		return expectedOk && actualOk && expectedResult.Operator == "==" && actual == expected
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
