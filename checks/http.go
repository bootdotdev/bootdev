package checks

import (
	"bytes"
	"encoding/json"
	"errors"
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
	Err            string      `json:"-"`
	RequestHeaders http.Header `json:"-"`
	StatusCode     int
	Headers        map[string]string
	BodyString     string
}

func HttpTest(
	lesson api.Lesson,
	baseURL *string,
) (
	responses []HttpTestResult,
	finalBaseURL string,
) {
	data := lesson.Lesson.LessonDataHTTPTests
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

		if request.Request.Actions.DelayRequestByMs != nil {
			time.Sleep(time.Duration(*request.Request.Actions.DelayRequestByMs) * time.Millisecond)
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
			RequestHeaders: r.Header,
			StatusCode:     resp.StatusCode,
			Headers:        headers,
			BodyString:     string(body),
		}
		parseVariables(body, request.ResponseVariables, variables)
	}
	return responses, finalBaseURL
}

func parseVariables(body []byte, vardefs []api.ResponseVariable, variables map[string]string) error {
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
