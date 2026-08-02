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
	ch := make(chan tea.Msg, 1)

	err := applySubmissionEvent(data, api.LessonSubmissionEvent{
		ResultSlug: api.VerificationResultSlugSystemError,
	}, ch)
	if err == nil || !strings.Contains(err.Error(), "system error") {
		t.Fatalf("applySubmissionEvent() error = %v, want system error", err)
	}

	select {
	case msg := <-ch:
		t.Fatalf("system error unexpectedly emitted result message: %#v", msg)
	default:
	}
}
