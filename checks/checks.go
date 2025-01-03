package checks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	api "github.com/bootdotdev/bootdev/client"
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
	if baseURL == "" && requestStep.Request.FullURL == "" {
		cobra.CheckErr("no base URL or full URL provided")
	}

	finalBaseURL := strings.TrimSuffix(baseURL, "/")
	interpolatedPath := InterpolateVariables(requestStep.Request.Path, variables)
	completeURL := fmt.Sprintf("%s%s", finalBaseURL, interpolatedPath)
	if requestStep.Request.FullURL != "" {
		completeURL = InterpolateVariables(requestStep.Request.FullURL, variables)
	}

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
		result = api.HTTPRequestResult{Err: "Failed to fetch"}
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

	parseVariables(body, requestStep.ResponseVariables, variables)

	result = api.HTTPRequestResult{
		StatusCode:      resp.StatusCode,
		ResponseHeaders: headers,
		BodyString:      truncateAndStringifyBody(body),
		Variables:       variables,
		Request:         requestStep,
	}
	return result
}

func CLIChecks(cliData api.CLIData, submitBaseURL *string) (results []api.CLIStepResult) {
	client := &http.Client{}
	variables := make(map[string]string)
	results = make([]api.CLIStepResult, len(cliData.Steps))

	// use cli arg url if specified or default lesson data url
	baseURL := ""
	if submitBaseURL != nil && *submitBaseURL != "" {
		baseURL = *submitBaseURL
	} else if cliData.BaseURL != nil && *cliData.BaseURL != "" {
		baseURL = *cliData.BaseURL
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
