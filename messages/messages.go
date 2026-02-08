package messages

import api "github.com/bootdotdev/bootdev/client"

type StartStepMsg struct {
	ResponseVariables []api.HTTPRequestResponseVariable
	CMD               string
	URL               string
	Method            string
	TmdlQuery         *string
}

type StartTestMsg struct {
	Text string
}

type ResolveTestMsg struct {
	StepIndex int
	TestIndex int
	Passed    *bool
}

type DoneStepMsg struct {
	Failure *api.VerificationResultStructuredErrCLI
}

type ResolveStepMsg struct {
	Index  int
	Passed *bool
	Result *api.CLIStepResult
}

type SleepMsg struct {
	DurationMs int
}
