package checks

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/itchyny/gojq"
)

type HttpTestError struct {
	FetchErr string
}

type HttpTestResult struct {
	StatusCode int
	Headers    map[string]string
	BodyString string
}

func HttpTest(assignment api.Assignment, port int) []any {
	data := assignment.Assignment.AssignmentDataHTTPTests
	client := &http.Client{}
	variables := make(map[string]string)
	responses := make([]any, len(data.HttpTests.Requests))
	for i, request := range data.HttpTests.Requests {
		req := request.Request

		// TODO: response variable interpolation
		r, err := http.NewRequest(req.Method, fmt.Sprintf("http://localhost:%d%s",
			port, req.Path), bytes.NewBuffer([]byte{}))
		if err != nil {
			responses[i] = HttpTestError{FetchErr: "Failed to create request"}
			continue
		}

		resp, err := client.Do(r)
		if err != nil {
			responses[i] = HttpTestError{FetchErr: "Failed to fetch"}
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			responses[i] = HttpTestError{FetchErr: "Failed to read response body"}
			continue
		}
		result := HttpTestResult{StatusCode: resp.StatusCode, Headers: make(map[string]string), BodyString: string(body)}
		for _, t := range request.Tests {
			if t.HeadersContain != nil {
				h := resp.Header.Get(t.HeadersContain.Key)
				if h != "" {
					result.Headers[t.HeadersContain.Key] = h
				}
			}
		}
		responses[i] = result
		parseVariables(body, request.ResponseVariables, &variables)

		if req.Actions.DelayRequestByMs != nil {
			time.Sleep(time.Duration(*req.Actions.DelayRequestByMs) * time.Millisecond)
		}
	}
	return responses
}

func parseVariables(body []byte, vardefs []api.ResponseVariable, variables *map[string]string) {
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
				(*variables)[vardef.Name] = str
				// TODO: remove this
				fmt.Println("parsed variable " + vardef.Name + " " + str)
			}
		}
	}
}
