package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type RunContext struct {
	Action      string
	LessonTitle string
	BaseURL     string
}

func renderRunHeader(context RunContext) string {
	if context.LessonTitle == "" {
		return ""
	}
	var str strings.Builder
	action := context.Action
	if action == "" {
		action = "Running lesson:"
	}
	fmt.Fprintln(&str, action)
	fmt.Fprintln(&str, lipgloss.NewStyle().Bold(true).Render(context.LessonTitle))
	if context.BaseURL != "" {
		fmt.Fprintf(&str, "%s %s\n", secondaryStyle.Render("Target:"), context.BaseURL)
	}
	str.WriteByte('\n')
	return str.String()
}
