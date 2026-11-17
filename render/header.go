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

func RenderLessonTitle(title string) string {
	return lipgloss.NewStyle().Bold(true).Render(title)
}

func renderRunHeader(context RunContext) string {
	var str strings.Builder
	if context.LessonTitle != "" {
		fmt.Fprintln(&str, context.Action)
		fmt.Fprintln(&str, RenderLessonTitle(context.LessonTitle))
	}
	if context.BaseURL != "" {
		if context.BaseURLOverridden {
			fmt.Fprintf(&str, "%s %s\n", warningStyle.Render("Using overridden base_url:"), context.BaseURL)
			fmt.Fprintln(&str, informationStyle.Render("You can reset to the default with `bootdev config base_url --reset`"))
		} else {
			fmt.Fprintf(&str, "%s %s\n", secondaryStyle.Render("base_url:"), context.BaseURL)
		}
	}
	if str.Len() > 0 {
		str.WriteByte('\n')
	}
	return str.String()
}
