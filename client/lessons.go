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
	StatusCode       *int
	BodyContains     *string
	BodyContainsNone *string
	HeadersContain   *HTTPTestHeader
	JSONValue        *HTTPTestJSONValue
}

type OperatorType string

const (
	OpEquals      OperatorType = "eq"
	OpGreaterThan OperatorType = "gt"
	OpContains    OperatorType = "contains"
	OpNotContains OperatorType = "not_contains"
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

type LessonDataHTTPTests struct {
	HttpTests struct {
		BaseURL             *string
		ContainsCompleteDir bool
		Requests            []LessonDataHTTPTestsRequest
	}
}

type LessonDataHTTPTestsRequest struct {
	ResponseVariables []ResponseVariable
	Tests             []HTTPTest
	Request           struct {
		FullURL   string // overrides BaseURL and Path if set
		Path      string
		BasicAuth *struct {
			Username string
			Password string
		}
		Headers  map[string]string
		BodyJSON map[string]interface{}
		Method   string
		Actions  struct {
			DelayRequestByMs *int32
		}
	}
}

type CLICommandTestCase struct {
	ExitCode           *int
	StdoutContainsAll  []string
	StdoutContainsNone []string
	StdoutLinesGt      *int
}

type LessonDataCLICommand struct {
	CLICommandData struct {
		Commands []struct {
			Command string
			Tests   []CLICommandTestCase
		}
	}
}

type LessonDataCLI struct {
	// Readme string
	CLIData CLIData
}

type CLIData struct {
	// ContainsCompleteDir bool
	BaseURL *string
	Steps   []struct {
		CLICommand  *CLIStepCLICommand
		HTTPRequest *CLIStepHTTPRequest
	}
}

type CLIStepCLICommand struct {
	Command string
	Tests   []CLICommandTestCase
}

type CLIStepHTTPRequest struct {
	ResponseVariables []ResponseVariable
	Tests             []HTTPTest
	Request           struct {
		Method    string
		Path      string
		FullURL   string // overrides BaseURL and Path if set
		Headers   map[string]string
		BodyJSON  map[string]interface{}
		BasicAuth *struct {
			Username string
			Password string
		}
		Actions struct {
			DelayRequestByMs *int32
		}
	}
}

type Lesson struct {
	Lesson struct {
		Type                 string
		LessonDataHTTPTests  *LessonDataHTTPTests
		LessonDataCLICommand *LessonDataCLICommand
		LessonDataCLI        *LessonDataCLI
	}
}

func FetchLesson(uuid string) (*Lesson, error) {
	resp, err := fetchWithAuth("GET", "/v1/static/lessons/"+uuid)
	if err != nil {
		return nil, err
	}

	var data Lesson
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

func SubmitHTTPTestLesson(uuid string, results any) (*HTTPTestValidationError, error) {
	bytes, err := json.Marshal(submitHTTPTestRequest{ActualHTTPRequests: results})
	if err != nil {
		return nil, err
	}
	resp, code, err := fetchWithAuthAndPayload("POST", "/v1/lessons/"+uuid+"/http_tests", bytes)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("failed to submit HTTP tests. code: %v: %s", code, string(resp))
	}
	var failure HTTPTestValidationError
	err = json.Unmarshal(resp, &failure)
	if err != nil || failure.ErrorMessage == nil {
		return nil, nil
	}
	return &failure, nil
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
	ExitCode     int
	FinalCommand string `json:"-"`
	Stdout       string
	Variables    map[string]string
}

func SubmitCLICommandLesson(uuid string, results []CLICommandResult) (*StructuredErrCLICommand, error) {
	bytes, err := json.Marshal(submitCLICommandRequest{CLICommandResults: results})
	if err != nil {
		return nil, err
	}
	resp, code, err := fetchWithAuthAndPayload("POST", "/v1/lessons/"+uuid+"/cli_command", bytes)
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

type HTTPRequestResult struct {
	Err             string `json:"-"`
	StatusCode      int
	ResponseHeaders map[string]string
	BodyString      string
	Variables       map[string]string
	Request         CLIStepHTTPRequest
}

type CLIStepResult struct {
	CLICommandResult  *CLICommandResult
	HTTPRequestResult *HTTPRequestResult
}

type lessonSubmissionCLI struct {
	CLIResults []CLIStepResult
}

type StructuredErrCLI struct {
	ErrorMessage    string `json:"Error"`
	FailedStepIndex int    `json:"FailedStepIndex"`
	FailedTestIndex int    `json:"FailedTestIndex"`
}

func SubmitCLILesson(uuid string, results []CLIStepResult) (*StructuredErrCLI, error) {
	bytes, err := json.Marshal(lessonSubmissionCLI{CLIResults: results})
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("/v1/lessons/%v/", uuid)
	resp, code, err := fetchWithAuthAndPayload("POST", endpoint, bytes)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("failed to submit CLI lesson (code: %v): %s", code, string(resp))
	}
	var failure StructuredErrCLI
	err = json.Unmarshal(resp, &failure)
	if err != nil || failure.ErrorMessage == "" {
		// this is ok - it means we had success
		return nil, nil
	}
	return &failure, nil
}
