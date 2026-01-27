package checks

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"net/http"
	"regexp"
	"strings"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/goccy/go-json"
	"github.com/spf13/cobra"
)

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

func InterpolateVariables(template string, vars map[string]string) string {
	r := regexp.MustCompile(`\$\{([^}]+)\}`)
	return r.ReplaceAllStringFunc(template, func(m string) string {
		// Extract the key from the match, which is in the form ${key}
		key := strings.TrimSuffix(strings.TrimPrefix(m, "${"), "}")
		if val, ok := vars[key]; ok {
			return val
		}
		return m
	})
}
