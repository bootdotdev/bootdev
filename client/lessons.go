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
	BaseURLDefault          string    `yaml:"baseURLDefault"`
	Steps                   []CLIStep `yaml:"steps"`
	AllowedOperatingSystems []string  `yaml:"allowedOperatingSystems"`
}

type CLIStep struct {
	CLICommand      *CLIStepCLICommand  `yaml:"cliCommand"`
	HTTPRequest     *CLIStepHTTPRequest `yaml:"httpRequest"`
	NoPenaltyOnFail bool                `yaml:"noPenaltyOnFail"`
}

type CLIStepCLICommand struct {
	Command          string           `yaml:"command"`
	Tests            []CLICommandTest `yaml:"tests"`
	SleepAfterMs     *int             `yaml:"sleepAfterMs"`
	StdoutFilterTmdl *string          `yaml:"stdoutFilterTmdl"`
}

type CLICommandTest struct {
	ExitCode           *int          `yaml:"exitCode"`
	StdoutContainsAll  []string      `yaml:"stdoutContainsAll"`
	StdoutContainsNone []string      `yaml:"stdoutContainsNone"`
	StdoutLinesGt      *int          `yaml:"stdoutLinesGt"`
	StdoutJq           *StdoutJqTest `yaml:"stdoutJq"`
}

type StdoutJqTest struct {
	InputMode       string             `yaml:"inputMode"` // "json" or "jsonl"
	Query           string             `yaml:"query"`
	ExpectedResults []JqExpectedResult `yaml:"expectedResults"`
}

type JqExpectedResult struct {
	Type     JqValueType `yaml:"type"`
	Operator JqOperator  `yaml:"operator"`
	Value    any         `yaml:"value"`
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
	ResponseVariables       []HTTPRequestResponseVariable       `yaml:"responseVariables"`
	ResponseHeaderVariables []HTTPRequestResponseHeaderVariable `yaml:"responseHeaderVariables"`
	Tests                   []HTTPRequestTest                   `yaml:"tests"`
	Request                 HTTPRequest                         `yaml:"request"`
	SleepAfterMs            *int                                `yaml:"sleepAfterMs"`
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
	Method          string            `yaml:"method"`
	FullURL         string            `yaml:"fullURL"`
	Headers         map[string]string `yaml:"headers"`
	BodyJSON        map[string]any    `yaml:"bodyJSON"`
	BodyForm        map[string]string `yaml:"bodyForm"`
	FollowRedirects *bool             `yaml:"followRedirects"`

	BasicAuth *HTTPBasicAuth `yaml:"basicAuth"`
}

type HTTPBasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type HTTPRequestResponseVariable struct {
	Name      string `yaml:"name"`
	Path      string `yaml:"path"`
	BodyRegex string `yaml:"bodyRegex"`
}

type HTTPRequestResponseHeaderVariable struct {
	Name   string `yaml:"name"`
	Header string `yaml:"header"`
	Regex  string `yaml:"regex"`
}

// HTTPRequestTest should have only one field set
type HTTPRequestTest struct {
	StatusCode       *int                      `yaml:"statusCode"`
	BodyContains     *string                   `yaml:"bodyContains"`
	BodyContainsNone *string                   `yaml:"bodyContainsNone"`
	HeadersContain   *HTTPRequestTestHeader    `yaml:"headersContain"`
	TrailersContain  *HTTPRequestTestHeader    `yaml:"trailersContain"`
	JSONValue        *HTTPRequestTestJSONValue `yaml:"jsonValue"`
}

type HTTPRequestTestHeader struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type HTTPRequestTestJSONValue struct {
	Path        string       `yaml:"path"`
	Operator    OperatorType `yaml:"operator"`
	IntValue    *int         `yaml:"intValue"`
	StringValue *string      `yaml:"stringValue"`
	BoolValue   *bool        `yaml:"boolValue"`
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

type XPBreakdownItem struct {
	Name    string
	Percent float64
	XP      int
}

type LessonSubmissionEvent struct {
	ResultSlug       VerificationResultSlug
	StructuredErrCLI *StructuredErrCLI
	XPReward         int
	XPBreakdown      []XPBreakdownItem
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
