package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

var green lipgloss.Style
var red lipgloss.Style
var gray lipgloss.Style
var borderBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

type testModel struct {
	text     string
	passed   *bool
	finished bool
}

type startTestMsg struct {
	text string
}

type resolveTestMsg struct {
	index  int
	passed *bool
}

func renderTestHeader(header string, spinner spinner.Model, isFinished bool, isSubmit bool, passed *bool) string {
	var cmdStr string
	if !isFinished {
		cmdStr += fmt.Sprintf("%s %s", spinner.View(), header)
	} else if !isSubmit {
		cmdStr += header
	} else if passed == nil {
		cmdStr += gray.Render(fmt.Sprintf("?  %s", header))
	} else if *passed {
		cmdStr += green.Render(fmt.Sprintf("✓  %s", header))
	} else {
		cmdStr += red.Render(fmt.Sprintf("X  %s", header))
	}
	box := borderBox.Render(fmt.Sprintf(" %s ", cmdStr))
	sliced := strings.Split(box, "\n")
	sliced[2] = strings.Replace(sliced[2], "─", "┬", 1)
	return strings.Join(sliced, "\n")
}

func renderTests(tests []testModel, spinner string) string {
	var str string
	for _, test := range tests {
		var testStr string
		if !test.finished {
			testStr += fmt.Sprintf("  %s %s", spinner, test.text)
		} else if test.passed == nil {
			testStr += gray.Render(fmt.Sprintf("  ?  %s", test.text))
		} else if *test.passed {
			testStr += green.Render(fmt.Sprintf("  ✓  %s", test.text))
		} else {
			testStr += red.Render(fmt.Sprintf("  X  %s", test.text))
		}
		edges := " ├─"
		for i := 0; i < lipgloss.Height(testStr)-1; i++ {
			edges += "\n │ "
		}
		testStr = lipgloss.JoinHorizontal(lipgloss.Top, edges, testStr)
		str = lipgloss.JoinVertical(lipgloss.Left, str, testStr)
	}
	str += "\n"
	return str
}
