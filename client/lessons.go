package api

import (
	"encoding/json"
	"fmt"
)

type Lesson struct {
	Lesson struct {
		Type          string
		LessonDataCLI *LessonDataCLI
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
	Tests   []CLICommandTest
}

type CLICommandTest struct {
	ExitCode           *int
	StdoutContainsAll  []string
	StdoutContainsNone []string
	StdoutLinesGt      *int
}

type CLIStepHTTPRequest struct {
	ResponseVariables []HTTPRequestResponseVariable
	Tests             []HTTPRequestTest
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

type HTTPRequestResponseVariable struct {
	Name string
	Path string
}

// Only one of these fields should be set
type HTTPRequestTest struct {
	StatusCode       *int
	BodyContains     *string
	BodyContainsNone *string
	HeadersContain   *HTTPRequestTestHeader
	JSONValue        *HTTPRequestTestJSONValue
}

type HTTPRequestTestHeader struct {
	Key   string
	Value string
}

type HTTPRequestTestJSONValue struct {
	Path        string
	Operator    OperatorType
	IntValue    *int
	StringValue *string
	BoolValue   *bool
}

type OperatorType string

const (
	OpEquals      OperatorType = "eq"
	OpGreaterThan OperatorType = "gt"
	OpContains    OperatorType = "contains"
	OpNotContains OperatorType = "not_contains"
)

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

type CLIStepResult struct {
	CLICommandResult  *CLICommandResult
	HTTPRequestResult *HTTPRequestResult
}

type CLICommandResult struct {
	ExitCode     int
	FinalCommand string `json:"-"`
	Stdout       string
	Variables    map[string]string
}

type HTTPRequestResult struct {
	Err             string `json:"-"`
	StatusCode      int
	ResponseHeaders map[string]string
	BodyString      string
	Variables       map[string]string
	Request         CLIStepHTTPRequest
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
