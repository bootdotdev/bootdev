package checks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
)

func runCLICommand(command api.CLIStepCLICommand, variables map[string]string) (result api.CLICommandResult) {
	finalCommand := InterpolateVariables(command.Command, variables)
	result.FinalCommand = finalCommand

	cmd := exec.Command("sh", "-c", finalCommand)
	cmd.Env = append(os.Environ(), "LANG=en_US.UTF-8")
	b, err := cmd.CombinedOutput()
	if ee, ok := err.(*exec.ExitError); ok {
		result.ExitCode = ee.ExitCode()
	} else if err != nil {
		result.ExitCode = -2
	}
	result.Stdout = strings.TrimRight(string(b), " \n\t\r")
	result.Variables = variables
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

	if requestStep.Request.Actions.DelayRequestByMs != nil {
		time.Sleep(time.Duration(*requestStep.Request.Actions.DelayRequestByMs) * time.Millisecond)
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
		Variables:        variables,
		Request:          requestStep,
	}
	return result
}

func CLIChecks(cliData api.CLIData, overrideBaseURL string) (results []api.CLIStepResult) {
	client := &http.Client{}
	variables := make(map[string]string)
	results = make([]api.CLIStepResult, len(cliData.Steps))

	if cliData.BaseURLDefault == api.BaseURLOverrideRequired && overrideBaseURL == "" {
		cobra.CheckErr("lesson requires a base URL override - bootdev configure base_url <url>")
	}

	// prefer overrideBaseURL if provided, otherwise use BaseURLDefault
	baseURL := overrideBaseURL
	if overrideBaseURL == "" {
		baseURL = cliData.BaseURLDefault
	}

	for i, step := range cliData.Steps {
		switch {
		case step.CLICommand != nil:
			result := runCLICommand(*step.CLICommand, variables)
			results[i].CLICommandResult = &result
		case step.HTTPRequest != nil:
			result := runHTTPRequest(client, baseURL, variables, *step.HTTPRequest)
			results[i].HTTPRequestResult = &result
			if result.Variables != nil {
				variables = result.Variables
			}
		default:
			cobra.CheckErr("unable to run lesson: missing step")
		}
	}
	return results
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
		val, err := valFromJQPath(vardef.Path, string(body))
		if err != nil {
			return err
		}
		variables[vardef.Name] = fmt.Sprintf("%v", val)
	}
	return nil
}

func valFromJQPath(path string, jsn string) (any, error) {
	vals, err := valsFromJQPath(path, jsn)
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

func valsFromJQPath(path string, jsn string) ([]any, error) {
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
