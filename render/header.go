package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type RunContext struct {
	Action            string
	LessonTitle       string
	BaseURL           string
	BaseURLOverridden bool
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
		if context.BaseURLOverridden {
			fmt.Fprintf(&str, "%s %s\n", warningStyle.Render("Using overridden base_url:"), context.BaseURL)
			fmt.Fprintln(&str, informationStyle.Render("You can reset to the default with `bootdev config base_url --reset`"))
		} else {
			fmt.Fprintf(&str, "%s %s\n", secondaryStyle.Render("base_url:"), context.BaseURL)
		}
	}
	str.WriteByte('\n')
	return str.String()
}
