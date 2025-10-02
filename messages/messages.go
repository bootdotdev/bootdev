package messages

import api "github.com/bootdotdev/bootdev/client"

type StartStepMsg struct {
	ResponseVariables []api.HTTPRequestResponseVariable
	CMD               string
	URL               string
	Method            string
}

type StartTestMsg struct {
	Text string
}

type ResolveTestMsg struct {
	Index  int
	Passed *bool
}

type DoneStepMsg struct {
	Failure *api.VerificationResultStructuredErrCLI
}

type ResolveStepMsg struct {
	Index  int
	Passed *bool
	Result *api.CLIStepResult
}
