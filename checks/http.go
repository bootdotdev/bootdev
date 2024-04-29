package checks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
)

type HttpTestResult struct {
	Err        string `json:"-"`
	StatusCode int
	Headers    map[string]string
	BodyString string
}

func HttpTest(
	assignment api.Assignment,
	baseURL *string,
) (
	responses []HttpTestResult,
	finalBaseURL string,
) {
	data := assignment.Assignment.AssignmentDataHTTPTests
	client := &http.Client{}
	variables := make(map[string]string)
	responses = make([]HttpTestResult, len(data.HttpTests.Requests))
	for i, request := range data.HttpTests.Requests {
		if baseURL != nil && *baseURL != "" {
			finalBaseURL = *baseURL
		} else if data.HttpTests.BaseURL != nil {
			finalBaseURL = *data.HttpTests.BaseURL
		} else {
			cobra.CheckErr("no base URL provided")
		}
		finalBaseURL = strings.TrimSuffix(finalBaseURL, "/")

		var r *http.Request
		if request.Request.BodyJSON != nil {
			dat, err := json.Marshal(request.Request.BodyJSON)
			cobra.CheckErr(err)
			r, err = http.NewRequest(request.Request.Method, fmt.Sprintf("%s%s",
				finalBaseURL, request.Request.Path), bytes.NewBuffer(dat))
			if err != nil {
				cobra.CheckErr("Failed to create request")
			}
		} else {
			var err error
			r, err = http.NewRequest(request.Request.Method, fmt.Sprintf("%s%s",
				finalBaseURL, request.Request.Path), nil)
			if err != nil {
				cobra.CheckErr("Failed to create request")
			}
		}

		for k, v := range request.Request.Headers {
			r.Header.Add(k, interpolateVariables(v, variables))
		}

		if request.Request.BasicAuth != nil {
			r.SetBasicAuth(request.Request.BasicAuth.Username, request.Request.BasicAuth.Password)
		}
		resp, err := client.Do(r)
		if err != nil {
			responses[i] = HttpTestResult{Err: "Failed to fetch"}
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			responses[i] = HttpTestResult{Err: "Failed to read response body"}
			continue
		}
		headers := make(map[string]string)
		for k, v := range resp.Header {
			headers[k] = strings.Join(v, ",")
		}
		responses[i] = HttpTestResult{
			StatusCode: resp.StatusCode,
			Headers:    headers,
			BodyString: string(body),
		}
		parseVariables(body, request.ResponseVariables, variables)

		if request.Request.Actions.DelayRequestByMs != nil {
			time.Sleep(time.Duration(*request.Request.Actions.DelayRequestByMs) * time.Millisecond)
		}
	}
	return responses, finalBaseURL
}

func parseVariables(body []byte, vardefs []api.ResponseVariable, variables map[string]string) {
	for _, vardef := range vardefs {
		query, err := gojq.Parse(vardef.Path)
		if err != nil {
			continue
		}
		code, err := gojq.Compile(query)
		if err != nil {
			continue
		}
		iter := code.Run(body)
		if value, ok := iter.Next(); ok {
			if str, ok := value.(string); ok {
				variables[vardef.Name] = str
			}
		}
	}
}

func interpolateVariables(template string, vars map[string]string) string {
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
