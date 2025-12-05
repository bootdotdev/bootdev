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

const BaseURLOverrideRequired = "override"

type CLIData struct {
	// ContainsCompleteDir bool
	BaseURLDefault          string
	Steps                   []CLIStep
	AllowedOperatingSystems []string
}

type CLIStep struct {
	CLICommand  *CLIStepCLICommand
	HTTPRequest *CLIStepHTTPRequest
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
	Request           HTTPRequest
}

const BaseURLPlaceholder = "${baseURL}"

type HTTPRequest struct {
	Method   string
	FullURL  string
	Headers  map[string]string
	BodyJSON map[string]any

	BasicAuth *HTTPBasicAuth
	Actions   HTTPActions
}

type HTTPBasicAuth struct {
	Username string
	Password string
}

type HTTPActions struct {
	DelayRequestByMs *int
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
	TrailersContain  *HTTPRequestTestHeader
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
	resp, err := fetchWithAuth("GET", "/v1/lessons/"+uuid)
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
	Err              string `json:"-"`
	StatusCode       int
	ResponseHeaders  map[string]string
	ResponseTrailers map[string]string
	BodyString       string
	Variables        map[string]string
	Request          CLIStepHTTPRequest
}

type lessonSubmissionCLI struct {
	CLIResults []CLIStepResult
}

type verificationResult struct {
	ResultSlug string
	// user friendly message to put in the toast
	ResultMessage string
	// only present if the lesson is an CLI type
	StructuredErrCLI *VerificationResultStructuredErrCLI
}

type VerificationResultStructuredErrCLI struct {
	ErrorMessage    string `json:"Error"`
	FailedStepIndex int    `json:"FailedStepIndex"`
	FailedTestIndex int    `json:"FailedTestIndex"`
}

func SubmitCLILesson(uuid string, results []CLIStepResult) (*VerificationResultStructuredErrCLI, error) {
	bytes, err := json.Marshal(lessonSubmissionCLI{CLIResults: results})
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("/v1/lessons/%v/", uuid)
	resp, code, err := fetchWithAuthAndPayload("POST", endpoint, bytes)
	if err != nil {
		return nil, err
	}
	if code == 402 {
		return nil, fmt.Errorf("To run and submit the tests for this lesson, you must have an active Boot.dev membership\nhttps://boot.dev/pricing")
	}
	if code != 200 {
		return nil, fmt.Errorf("failed to submit CLI lesson (code %v): %s", code, string(resp))
	}

	result := verificationResult{}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return nil, err
	}
	return result.StructuredErrCLI, nil
}
