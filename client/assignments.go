package api

import (
	"encoding/json"
	"fmt"
)

type ResponseVariable struct {
	Name string
	Path string
}

// Only one of these fields should be set
type HTTPTest struct {
	StatusCode     *int
	BodyContains   *string
	HeadersContain *HTTPTestHeader
	JSONValue      *HTTPTestJSONValue
}

type OperatorType string

const (
	OpEquals      OperatorType = "eq"
	OpGreaterThan OperatorType = "gt"
)

type HTTPTestJSONValue struct {
	Path        string
	Operator    OperatorType
	IntValue    *int
	StringValue *string
	BoolValue   *bool
}

type HTTPTestHeader struct {
	Key   string
	Value string
}

type AssignmentDataHTTPTests struct {
	HttpTests struct {
		BaseURL             *string
		ContainsCompleteDir bool
		Requests            []struct {
			ResponseVariables []ResponseVariable
			Tests             []HTTPTest
			Request           struct {
				BasicAuth *struct {
					Username string
					Password string
				}
				Headers  map[string]string
				BodyJSON map[string]interface{}
				Method   string
				Path     string
				Actions  struct {
					DelayRequestByMs *int32
				}
			}
		}
	}
}

type CLICommandTestCase struct {
	ExitCode           *int
	StdoutContainsAll  []string
	StdoutContainsNone []string
	StdoutMatches      *string
	StdoutLinesGt      *int
}

type AssignmentDataCLICommand struct {
	CLICommandData struct {
		Commands []struct {
			Command string
			Tests   []CLICommandTestCase
		}
	}
}

type Assignment struct {
	Assignment struct {
		Type                     string
		AssignmentDataHTTPTests  *AssignmentDataHTTPTests
		AssignmentDataCLICommand *AssignmentDataCLICommand
	}
}

func FetchAssignment(uuid string) (*Assignment, error) {
	resp, err := fetchWithAuth("GET", "/v1/assignments/"+uuid)
	if err != nil {
		return nil, err
	}

	var data Assignment
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

type HTTPTestValidationError struct {
	ErrorMessage       *string `json:"Error"`
	FailedRequestIndex *int    `json:"FailedRequestIndex"`
	FailedTestIndex    *int    `json:"FailedTestIndex"`
}

type submitHTTPTestRequest struct {
	ActualHTTPRequests any `json:"actualHTTPRequests"`
}

func SubmitHTTPTestAssignment(uuid string, results any) error {
	bytes, err := json.Marshal(submitHTTPTestRequest{ActualHTTPRequests: results})
	if err != nil {
		return err
	}
	resp, code, err := fetchWithAuthAndPayload("POST", "/v1/assignments/"+uuid+"/http_tests", bytes)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("failed to submit HTTP tests. code: %v: %s", code, string(resp))
	}
	return nil
}

type submitCLICommandRequest struct {
	CLICommandResults []CLICommandResult `json:"cliCommandResults"`
}

type StructuredErrCLICommand struct {
	ErrorMessage       string `json:"Error"`
	FailedCommandIndex int    `json:"FailedCommandIndex"`
	FailedTestIndex    int    `json:"FailedTestIndex"`
}

type CLICommandResult struct {
	ExitCode int
	Stdout   string
}

func SubmitCLICommandAssignment(uuid string, results []CLICommandResult) (*StructuredErrCLICommand, error) {
	bytes, err := json.Marshal(submitCLICommandRequest{CLICommandResults: results})
	if err != nil {
		return nil, err
	}
	resp, code, err := fetchWithAuthAndPayload("POST", "/v1/assignments/"+uuid+"/cli_command", bytes)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("failed to submit CLI command tests. code: %v: %s", code, string(resp))
	}
	var failure StructuredErrCLICommand
	err = json.Unmarshal(resp, &failure)
	if err != nil || failure.ErrorMessage == "" {
		// this is ok - it means we had success
		return nil, nil
	}
	return &failure, nil
}
