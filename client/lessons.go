package api

import (
	"fmt"

	"github.com/goccy/go-json"
)

type Lesson struct {
	Lesson struct {
		Type          string
		LessonDataCLI *LessonDataCLI
	}
}

type LessonDataCLI struct {
	// Readme  string
	CLIData CLIData
}

const BaseURLOverrideRequired = "override"

type CLIData struct {
	// ContainsCompleteDir     bool
	BaseURLDefault          string
	Steps                   []CLIStep
	AllowedOperatingSystems []string
}

type CLIStep struct {
	CLICommand      *CLIStepCLICommand
	HTTPRequest     *CLIStepHTTPRequest
	NoPenaltyOnFail bool
}

type CLIStepCLICommand struct {
	Command          string
	Tests            []CLICommandTest
	SleepAfterMs     *int
	StdoutFilterTmdl *string
}

type CLICommandTest struct {
	ExitCode           *int
	StdoutContainsAll  []string
	StdoutContainsNone []string
	StdoutLinesGt      *int
	StdoutJq           *StdoutJqTest
}

type StdoutJqTest struct {
	InputMode       string // "json" or "jsonl"
	Query           string
	ExpectedResults []JqExpectedResult
}

type JqExpectedResult struct {
	Type     JqValueType
	Operator JqOperator
	Value    any
}

type (
	JqValueType string
	JqOperator  string // defined fully on backend
)

const (
	JqTypeString JqValueType = "string"
	JqTypeInt    JqValueType = "int"
	JqTypeBool   JqValueType = "bool"
)

type CLIStepHTTPRequest struct {
	ResponseVariables []HTTPRequestResponseVariable
	Tests             []HTTPRequestTest
	Request           HTTPRequest
	SleepAfterMs      *int
}

type Sleepable interface {
	GetSleepAfterMs() *int
}

func (c *CLIStepCLICommand) GetSleepAfterMs() *int {
	return c.SleepAfterMs
}

func (h *CLIStepHTTPRequest) GetSleepAfterMs() *int {
	return h.SleepAfterMs
}

const BaseURLPlaceholder = "${baseURL}"

type HTTPRequest struct {
	Method   string
	FullURL  string
	Headers  map[string]string
	BodyJSON map[string]any
	BodyForm map[string]string

	BasicAuth *HTTPBasicAuth
}

type HTTPBasicAuth struct {
	Username string
	Password string
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
	FinalCommand string            `json:"-"`
	Command      CLIStepCLICommand `json:"-"`
	Stdout       string
	Variables    map[string]string
	JqOutputs    []CLICommandJqOutput `json:"-"`
}

type CLICommandJqOutput struct {
	Query   string
	Results []string
	Error   string
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

type SubmissionDebugData struct {
	Endpoint           string
	RequestBody        string
	ResponseStatusCode int
	ResponseBody       string
}

type LessonSubmissionEvent struct {
	ResultSlug       VerificationResultSlug
	StructuredErrCLI *StructuredErrCLI
}

type StructuredErrCLI struct {
	ErrorMessage    string `json:"Error"`
	FailedStepIndex int    `json:"FailedStepIndex"`
	FailedTestIndex int    `json:"FailedTestIndex"`
}

type VerificationResultSlug string

const (
	// "noop" is for "noPenaltyOnFail" on the CLI type
	VerificationResultSlugNoop        VerificationResultSlug = "noop"
	VerificationResultSlugSystemError VerificationResultSlug = "system-error"
	VerificationResultSlugSuccess     VerificationResultSlug = "success"
	VerificationResultSlugFailure     VerificationResultSlug = "failure"
)

func SubmitCLILesson(uuid string, results []CLIStepResult, captureDebug bool) (LessonSubmissionEvent, SubmissionDebugData, error) {
	endpoint := fmt.Sprintf("/v1/lessons/%v/", uuid)
	debugData := SubmissionDebugData{Endpoint: endpoint}

	bytes, err := json.Marshal(lessonSubmissionCLI{CLIResults: results})
	if err != nil {
		return LessonSubmissionEvent{}, debugData, err
	}
	if captureDebug {
		debugData.RequestBody = string(bytes)
	}

	resp, code, err := fetchWithAuthAndPayload("POST", endpoint, bytes)
	debugData.ResponseStatusCode = code
	if captureDebug {
		debugData.ResponseBody = string(resp)
	}
	if err != nil {
		return LessonSubmissionEvent{}, debugData, err
	}
	if code == 402 {
		return LessonSubmissionEvent{}, debugData, fmt.Errorf("to run and submit the tests for this lesson, you must have an active Boot.dev membership\nhttps://boot.dev/pricing")
	}
	if code != 200 {
		return LessonSubmissionEvent{}, debugData, fmt.Errorf("failed to submit CLI lesson (code %v): %s", code, string(resp))
	}

	result := LessonSubmissionEvent{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return LessonSubmissionEvent{}, debugData, err
	}
	return result, debugData, nil
}
