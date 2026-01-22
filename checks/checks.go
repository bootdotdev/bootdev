package checks

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/goccy/go-json"
	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func runCLICommand(command api.CLIStepCLICommand, variables map[string]string) (result api.CLICommandResult) {
	finalCommand := InterpolateVariables(command.Command, variables)
	result.FinalCommand = finalCommand

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-Command", finalCommand)
	} else {
		cmd = exec.Command("sh", "-c", finalCommand)
	}

	cmd.Env = append(os.Environ(), "LANG=en_US.UTF-8")
	b, err := cmd.CombinedOutput()
	if ee, ok := err.(*exec.ExitError); ok {
		result.ExitCode = ee.ExitCode()
	} else if err != nil {
		result.ExitCode = -2
	}
	result.Stdout = strings.TrimRight(string(b), " \n\t\r")
	result.Variables = maps.Clone(variables)
	return result
}

func runHTTPRequest(
	client *http.Client,
	baseURL string,
	variables map[string]string,
	requestStep api.CLIStepHTTPRequest,
) (
	result api.HTTPRequestResult,
) {
	finalBaseURL := strings.TrimSuffix(baseURL, "/")
	interpolatedURL := InterpolateVariables(requestStep.Request.FullURL, variables)
	completeURL := strings.Replace(interpolatedURL, api.BaseURLPlaceholder, finalBaseURL, 1)

	var req *http.Request
	if requestStep.Request.BodyJSON != nil {
		dat, err := json.Marshal(requestStep.Request.BodyJSON)
		cobra.CheckErr(err)
		interpolatedBodyJSONStr := InterpolateVariables(string(dat), variables)
		req, err = http.NewRequest(requestStep.Request.Method, completeURL,
			bytes.NewBuffer([]byte(interpolatedBodyJSONStr)),
		)
		if err != nil {
			cobra.CheckErr("Failed to create request")
		}
		req.Header.Add("Content-Type", "application/json")
	} else {
		var err error
		req, err = http.NewRequest(requestStep.Request.Method, completeURL, nil)
		if err != nil {
			cobra.CheckErr("Failed to create request")
		}
	}

	for k, v := range requestStep.Request.Headers {
		req.Header.Add(k, InterpolateVariables(v, variables))
	}

	if requestStep.Request.BasicAuth != nil {
		req.SetBasicAuth(requestStep.Request.BasicAuth.Username, requestStep.Request.BasicAuth.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		errString := fmt.Sprintf("Failed to fetch: %s", err.Error())
		result = api.HTTPRequestResult{Err: errString}
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result = api.HTTPRequestResult{Err: "Failed to read response body"}
		return result
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = strings.Join(v, ",")
	}

	trailers := make(map[string]string)
	for k, v := range resp.Trailer {
		trailers[k] = strings.Join(v, ",")
	}

	parseVariables(body, requestStep.ResponseVariables, variables)

	result = api.HTTPRequestResult{
		StatusCode:       resp.StatusCode,
		ResponseHeaders:  headers,
		ResponseTrailers: trailers,
		BodyString:       truncateAndStringifyBody(body),
		Variables:        maps.Clone(variables),
		Request:          requestStep,
	}
	return result
}

func CLIChecks(cliData api.CLIData, overrideBaseURL string, ch chan tea.Msg) (results []api.CLIStepResult) {
	client := &http.Client{}
	variables := make(map[string]string)
	results = make([]api.CLIStepResult, len(cliData.Steps))

	if cliData.BaseURLDefault == api.BaseURLOverrideRequired && overrideBaseURL == "" {
		cobra.CheckErr("lesson requires a base URL override: `bootdev configure base_url <url>`")
	}

	// prefer overrideBaseURL if provided, otherwise use BaseURLDefault
	baseURL := overrideBaseURL
	if overrideBaseURL == "" {
		baseURL = cliData.BaseURLDefault
	}

	for i, step := range cliData.Steps {
		// This is the magic of the initial message sent before executing the test
		if step.CLICommand != nil {
			ch <- messages.StartStepMsg{CMD: step.CLICommand.Command}
		} else if step.HTTPRequest != nil {
			finalBaseURL := baseURL
			overrideURL := viper.GetString("override_base_url")
			if overrideURL != "" {
				finalBaseURL = overrideURL
			}
			fullURL := strings.Replace(step.HTTPRequest.Request.FullURL, api.BaseURLPlaceholder, finalBaseURL, 1)
			interpolatedURL := InterpolateVariables(fullURL, variables)

			ch <- messages.StartStepMsg{
				URL:               interpolatedURL,
				Method:            step.HTTPRequest.Request.Method,
				ResponseVariables: step.HTTPRequest.ResponseVariables,
			}
		}

		switch {
		case step.CLICommand != nil:
			result := runCLICommand(*step.CLICommand, variables)
			result.JqOutputs = collectStdoutJqOutputs(*step.CLICommand, result)
			results[i].CLICommandResult = &result

			sendCLICommandResults(ch, *step.CLICommand, result, i)
			handleSleep(step.CLICommand, ch)

		case step.HTTPRequest != nil:
			result := runHTTPRequest(client, baseURL, variables, *step.HTTPRequest)
			results[i].HTTPRequestResult = &result
			sendHTTPRequestResults(ch, *step.HTTPRequest, result, i)
			handleSleep(step.HTTPRequest, ch)

		default:
			cobra.CheckErr("unable to run lesson: missing step")
		}
	}
	return results
}

func sendCLICommandResults(ch chan tea.Msg, cmd api.CLIStepCLICommand, result api.CLICommandResult, index int) {
	for _, test := range cmd.Tests {
		ch <- messages.StartTestMsg{Text: prettyPrintCLICommand(test, result.Variables)}
	}

	for j := range cmd.Tests {
		ch <- messages.ResolveTestMsg{
			StepIndex: index,
			TestIndex: j,
		}
	}

	ch <- messages.ResolveStepMsg{
		Index: index,
		Result: &api.CLIStepResult{
			CLICommandResult: &result,
		},
	}
}

func sendHTTPRequestResults(ch chan tea.Msg, req api.CLIStepHTTPRequest, result api.HTTPRequestResult, index int) {
	for _, test := range req.Tests {
		ch <- messages.StartTestMsg{Text: prettyPrintHTTPTest(test, result.Variables)}
	}

	for j := range req.Tests {
		ch <- messages.ResolveTestMsg{
			StepIndex: index,
			TestIndex: j,
		}
	}

	ch <- messages.ResolveStepMsg{
		Index: index,
		Result: &api.CLIStepResult{
			HTTPRequestResult: &result,
		},
	}
}

func ApplySubmissionResults(cliData api.CLIData, failure *api.VerificationResultStructuredErrCLI, ch chan tea.Msg) {
	for i, step := range cliData.Steps {
		stepPass := true
		isFailedStep := false
		if failure != nil {
			stepPass = i < failure.FailedStepIndex
			isFailedStep = i == failure.FailedStepIndex
		}

		ch <- messages.ResolveStepMsg{
			Index:  i,
			Passed: &stepPass,
		}

		if step.CLICommand != nil {
			for j := range step.CLICommand.Tests {
				if isFailedStep && j > failure.FailedTestIndex {
					break
				}

				testPass := stepPass || (isFailedStep && j < failure.FailedTestIndex)
				ch <- messages.ResolveTestMsg{
					StepIndex: i,
					TestIndex: j,
					Passed:    &testPass,
				}
			}
		}
		if step.HTTPRequest != nil {
			for j := range step.HTTPRequest.Tests {
				if isFailedStep && j > failure.FailedTestIndex {
					break
				}

				testPass := stepPass || (isFailedStep && j < failure.FailedTestIndex)
				ch <- messages.ResolveTestMsg{
					StepIndex: i,
					TestIndex: j,
					Passed:    &testPass,
				}
			}
		}

		if !stepPass {
			break
		}

	}
}

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

func prettyPrintCLICommand(test api.CLICommandTest, variables map[string]string) string {
	if test.ExitCode != nil {
		return fmt.Sprintf("Expect exit code %d", *test.ExitCode)
	}
	if test.StdoutLinesGt != nil {
		return fmt.Sprintf("Expect > %d lines on stdout", *test.StdoutLinesGt)
	}
	if test.StdoutContainsAll != nil {
		var str strings.Builder
		str.WriteString("Expect stdout to contain all of:")
		for _, contains := range test.StdoutContainsAll {
			interpolatedContains := InterpolateVariables(contains, variables)
			fmt.Fprintf(&str, "\n      - '%s'", interpolatedContains)
		}
		return str.String()
	}
	if test.StdoutContainsNone != nil {
		var str strings.Builder
		str.WriteString("Expect stdout to contain none of:")
		for _, containsNone := range test.StdoutContainsNone {
			interpolatedContainsNone := InterpolateVariables(containsNone, variables)
			fmt.Fprintf(&str, "\n      - '%s'", interpolatedContainsNone)
		}
		return str.String()
	}
	if test.StdoutJq != nil {
		return prettyPrintStdoutJqTest(*test.StdoutJq, variables)
	}
	return ""
}

func prettyPrintHTTPTest(test api.HTTPRequestTest, variables map[string]string) string {
	if test.StatusCode != nil {
		return fmt.Sprintf("Expecting status code: %d", *test.StatusCode)
	}
	if test.BodyContains != nil {
		interpolated := InterpolateVariables(*test.BodyContains, variables)
		return fmt.Sprintf("Expecting body to contain: %s", interpolated)
	}
	if test.BodyContainsNone != nil {
		interpolated := InterpolateVariables(*test.BodyContainsNone, variables)
		return fmt.Sprintf("Expecting JSON body to not contain: %s", interpolated)
	}
	if test.HeadersContain != nil {
		interpolatedKey := InterpolateVariables(test.HeadersContain.Key, variables)
		interpolatedValue := InterpolateVariables(test.HeadersContain.Value, variables)
		return fmt.Sprintf("Expecting headers to contain: '%s: %v'", interpolatedKey, interpolatedValue)
	}
	if test.TrailersContain != nil {
		interpolatedKey := InterpolateVariables(test.TrailersContain.Key, variables)
		interpolatedValue := InterpolateVariables(test.TrailersContain.Value, variables)
		return fmt.Sprintf("Expecting trailers to contain: '%s: %v'", interpolatedKey, interpolatedValue)
	}
	if test.JSONValue != nil {
		var val any
		switch {
		case test.JSONValue.IntValue != nil:
			val = *test.JSONValue.IntValue
		case test.JSONValue.StringValue != nil:
			val = *test.JSONValue.StringValue
		case test.JSONValue.BoolValue != nil:
			val = *test.JSONValue.BoolValue
		}

		var op string
		switch test.JSONValue.Operator {
		case api.OpEquals:
			op = "to be equal to"
		case api.OpGreaterThan:
			op = "to be greater than"
		case api.OpContains:
			op = "contains"
		case api.OpNotContains:
			op = "to not contain"
		}

		expecting := fmt.Sprintf("Expecting JSON at %v %s %v", test.JSONValue.Path, op, val)
		return InterpolateVariables(expecting, variables)
	}
	return ""
}

// truncateAndStringifyBody
// in some lessons we yeet the entire body up to the server, but we really shouldn't ever care
// about more than 100,000 stringified characters of it, so this protects against giant bodies
func truncateAndStringifyBody(body []byte) string {
	bodyString := string(body)
	const maxBodyLength = 1000000
	if len(bodyString) > maxBodyLength {
		bodyString = bodyString[:maxBodyLength]
	}
	return bodyString
}

func parseVariables(body []byte, vardefs []api.HTTPRequestResponseVariable, variables map[string]string) error {
	for _, vardef := range vardefs {
		val, err := valFromJqPath(vardef.Path, string(body))
		if err != nil {
			return err
		}
		variables[vardef.Name] = fmt.Sprintf("%v", val)
	}
	return nil
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

func InterpolateVariables(template string, vars map[string]string) string {
	r := regexp.MustCompile(`\$\{([^}]+)\}`)
	return r.ReplaceAllStringFunc(template, func(m string) string {
		// Extract the key from the match, which is in the form ${key}
		key := strings.TrimSuffix(strings.TrimPrefix(m, "${"), "}")
		if val, ok := vars[key]; ok {
			return val
		}
		return m // return the original placeholder if no substitution found
	})
}

func handleSleep(s api.Sleepable, ch chan tea.Msg) {
	sleepMs := s.GetSleepAfterMs()
	if sleepMs != nil && *sleepMs > 0 {
		ch <- messages.SleepMsg{DurationMs: *sleepMs}
		time.Sleep(time.Duration(*sleepMs) * time.Millisecond)
	}
}
