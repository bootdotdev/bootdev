package checks

import (
	"net/http"
	"strings"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

const lessonHTTPRequestTimeout = 30 * time.Second

func CLIChecks(cliData api.CLIData, overrideBaseURL string, send func(tea.Msg)) (results []api.CLIStepResult) {
	client := &http.Client{Timeout: lessonHTTPRequestTimeout}
	results = make([]api.CLIStepResult, len(cliData.Steps))

	if cliData.BaseURLDefault == api.BaseURLOverrideRequired && overrideBaseURL == "" {
		cobra.CheckErr("lesson requires a base URL override: `bootdev configure base_url <url>`")
	}

	baseURL := overrideBaseURL
	if overrideBaseURL == "" {
		baseURL = cliData.BaseURLDefault
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	variables := make(map[string]string)
	if baseURL != "" {
		variables["baseURL"] = baseURL
	}

	for i, step := range cliData.Steps {
		// This is the magic of the initial message sent before executing the test
		if step.CLICommand != nil {
			send(messages.StartStepMsg{
				Description:     step.Description,
				CMD:             step.CLICommand.Command,
				TmdlQuery:       step.CLICommand.StdoutFilterTmdl,
				NoPenaltyOnFail: step.NoPenaltyOnFail,
			})
		} else if step.HTTPRequest != nil {
			fullURL := strings.Replace(step.HTTPRequest.Request.FullURL, api.BaseURLPlaceholder, baseURL, 1)
			interpolatedURL := InterpolateVariables(fullURL, variables)

			send(messages.StartStepMsg{
				Description:     step.Description,
				URL:             interpolatedURL,
				Method:          step.HTTPRequest.Request.Method,
				NoPenaltyOnFail: step.NoPenaltyOnFail,
			})
		}

		switch {
		case step.CLICommand != nil:
			result := runCLICommand(*step.CLICommand, variables)
			result.JqOutputs = collectStdoutJqOutputs(*step.CLICommand, result)
			results[i].CLICommandResult = &result

			sendCLICommandResults(send, *step.CLICommand, result, i)
			handleSleep(step.CLICommand.SleepAfterMs, send)

		case step.HTTPRequest != nil:
			result := runHTTPRequest(client, baseURL, variables, *step.HTTPRequest)
			results[i].HTTPRequestResult = &result
			sendHTTPRequestResults(send, *step.HTTPRequest, result, i)
			handleSleep(step.HTTPRequest.SleepAfterMs, send)

		default:
			cobra.CheckErr("unable to run lesson: missing step")
		}
	}
	return results
}

func sendCLICommandResults(send func(tea.Msg), cmd api.CLIStepCLICommand, result api.CLICommandResult, index int) {
	for _, test := range cmd.Tests {
		send(messages.StartTestMsg{Text: prettyPrintCLICommand(test, result.Variables)})
	}

	for j := range cmd.Tests {
		send(messages.ResolveTestMsg{
			StepIndex: index,
			TestIndex: j,
		})
	}

	send(messages.ResolveStepMsg{
		Index: index,
		Result: &api.CLIStepResult{
			CLICommandResult: &result,
		},
	})
}

func sendHTTPRequestResults(send func(tea.Msg), req api.CLIStepHTTPRequest, result api.HTTPRequestResult, index int) {
	for _, test := range req.Tests {
		send(messages.StartTestMsg{Text: prettyPrintHTTPTest(test, result.Variables)})
	}

	for j := range req.Tests {
		send(messages.ResolveTestMsg{
			StepIndex: index,
			TestIndex: j,
		})
	}

	send(messages.ResolveStepMsg{
		Index: index,
		Result: &api.CLIStepResult{
			HTTPRequestResult: &result,
		},
	})
}

func ApplySubmissionResults(cliData api.CLIData, failure *api.StructuredErrCLI, send func(tea.Msg)) {
	for i, step := range cliData.Steps {
		stepPass := true
		isFailedStep := false
		if failure != nil {
			stepPass = i < failure.FailedStepIndex
			isFailedStep = i == failure.FailedStepIndex
		}

		send(messages.ResolveStepMsg{
			Index:  i,
			Passed: &stepPass,
		})

		if step.CLICommand != nil {
			for j := range step.CLICommand.Tests {
				if isFailedStep && j > failure.FailedTestIndex {
					break
				}

				testPass := stepPass || (isFailedStep && j < failure.FailedTestIndex)
				send(messages.ResolveTestMsg{
					StepIndex: i,
					TestIndex: j,
					Passed:    &testPass,
				})
			}
		}
		if step.HTTPRequest != nil {
			for j := range step.HTTPRequest.Tests {
				if isFailedStep && j > failure.FailedTestIndex {
					break
				}

				testPass := stepPass || (isFailedStep && j < failure.FailedTestIndex)
				send(messages.ResolveTestMsg{
					StepIndex: i,
					TestIndex: j,
					Passed:    &testPass,
				})
			}
		}

		if !stepPass {
			break
		}
	}
}

func handleSleep(sleepMs *int, send func(tea.Msg)) {
	if sleepMs != nil && *sleepMs > 0 {
		send(messages.SleepMsg{DurationMs: *sleepMs})
		time.Sleep(time.Duration(*sleepMs) * time.Millisecond)
	}
}
