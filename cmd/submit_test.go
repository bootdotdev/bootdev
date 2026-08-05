package cmd

import (
	"strings"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
	tea "github.com/charmbracelet/bubbletea"
)

func TestApplySubmissionEventRejectsSystemErrorWithoutMarkingStepsPassed(t *testing.T) {
	data := api.CLIData{Steps: []api.CLIStep{{
		CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{}}},
	}}}
	var sent []tea.Msg

	err := applySubmissionEvent(data, api.LessonSubmissionEvent{
		ResultSlug: api.VerificationResultSlugSystemError,
	}, func(msg tea.Msg) {
		sent = append(sent, msg)
	})
	if err == nil || !strings.Contains(err.Error(), "system error") {
		t.Fatalf("applySubmissionEvent() error = %v, want system error", err)
	}
	if len(sent) != 0 {
		t.Fatalf("system error unexpectedly emitted result messages: %#v", sent)
	}
}
