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
	cmdStr := renderTest(header, spinner.View(), isFinished, &isSubmit, passed)
	box := borderBox.Render(fmt.Sprintf(" %s ", cmdStr))
	sliced := strings.Split(box, "\n")
	sliced[2] = strings.Replace(sliced[2], "─", "┬", 1)
	return strings.Join(sliced, "\n") + "\n"
}

func renderTests(tests []testModel, spinner string) string {
	var str string
	for _, test := range tests {
		testStr := renderTest(test.text, spinner, test.finished, nil, test.passed)
		testStr = fmt.Sprintf("  %s", testStr)

		edges := " ├─"
		for i := 0; i < lipgloss.Height(testStr)-1; i++ {
			edges += "\n │ "
		}
		str += lipgloss.JoinHorizontal(lipgloss.Top, edges, testStr)
		str += "\n"
	}
	str += "\n"
	return str
}

func renderTest(text string, spinner string, isFinished bool, isSubmit *bool, passed *bool) string {
	testStr := ""
	if !isFinished {
		testStr += fmt.Sprintf("%s %s", spinner, text)
	} else if isSubmit != nil && !*isSubmit {
		testStr += text
	} else if passed == nil {
		testStr += gray.Render(fmt.Sprintf("?  %s", text))
	} else if *passed {
		testStr += green.Render(fmt.Sprintf("✓  %s", text))
	} else {
		testStr += red.Render(fmt.Sprintf("X  %s", text))
	}
	return testStr
}
