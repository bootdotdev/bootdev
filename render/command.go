package render

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/viper"
)

type doneCmdMsg struct {
	failure *api.StructuredErrCLICommand
}

type startCmdMsg struct {
	cmd string
}

type resolveCmdMsg struct {
	index   int
	passed  *bool
	results *api.CLICommandResult
}

type cmdModel struct {
	command  string
	passed   *bool
	results  *api.CLICommandResult
	finished bool
	tests    []testModel
}

type cmdRootModel struct {
	cmds      []cmdModel
	spinner   spinner.Model
	failure   *api.StructuredErrCLICommand
	isSubmit  bool
	success   bool
	finalized bool
	clear     bool
}

func initialModelCmd(isSubmit bool) cmdRootModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return cmdRootModel{
		spinner:  s,
		isSubmit: isSubmit,
		cmds:     []cmdModel{},
	}
}

func (m cmdRootModel) Init() tea.Cmd {
	green = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.green")))
	red = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.red")))
	gray = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.gray")))
	return m.spinner.Tick
}

func (m cmdRootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneCmdMsg:
		m.failure = msg.failure
		if m.failure == nil && m.isSubmit {
			m.success = true
		}
		m.clear = true
		return m, tea.Quit

	case startCmdMsg:
		m.cmds = append(m.cmds, cmdModel{command: fmt.Sprintf("Running: %s", msg.cmd), tests: []testModel{}})
		return m, nil

	case resolveCmdMsg:
		m.cmds[msg.index].passed = msg.passed
		m.cmds[msg.index].finished = true
		m.cmds[msg.index].results = msg.results
		return m, nil

	case startTestMsg:
		m.cmds[len(m.cmds)-1].tests = append(
			m.cmds[len(m.cmds)-1].tests,
			testModel{text: msg.text},
		)
		return m, nil

	case resolveTestMsg:
		m.cmds[len(m.cmds)-1].tests[msg.index].passed = msg.passed
		m.cmds[len(m.cmds)-1].tests[msg.index].finished = true
		return m, nil

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m cmdRootModel) View() string {
	if m.clear {
		return ""
	}
	s := m.spinner.View()
	var str string
	for _, cmd := range m.cmds {
		str += renderTestHeader(cmd.command, m.spinner, cmd.finished, m.isSubmit, cmd.passed)
		str += renderTests(cmd.tests, s)
		if cmd.results != nil && m.finalized {
			// render the results
			str += fmt.Sprintf("\n > Command exit code: %d\n", cmd.results.ExitCode)
			str += " > Command stdout:\n\n"
			sliced := strings.Split(cmd.results.Stdout, "\n")
			for _, s := range sliced {
				str += gray.Render(s) + "\n"
			}
		}
	}
	if m.failure != nil {
		str += red.Render("\n\nError: "+m.failure.ErrorMessage) + "\n\n"
	} else if m.success {
		str += "\n\n" + green.Render("All tests passed! 🎉") + "\n\n"
		str += green.Render("Return to your browser to continue with the next lesson.") + "\n\n"
	}
	return str
}

func prettyPrintCmd(test api.CLICommandTestCase) string {
	if test.ExitCode != nil {
		return fmt.Sprintf("Expect exit code %d", *test.ExitCode)
	}
	if test.StdoutLinesGt != nil {
		return fmt.Sprintf("Expect > %d lines on stdout", *test.StdoutLinesGt)
	}
	if test.StdoutMatches != nil {
		return fmt.Sprintf("Expect stdout to match '%s'", *test.StdoutMatches)
	}
	if test.StdoutContainsAll != nil {
		str := "Expect stdout to contain all of:"
		for _, thing := range test.StdoutContainsAll {
			str += fmt.Sprintf("\n      - '%s'", thing)
		}
		return str
	}
	if test.StdoutContainsNone != nil {
		str := "Expect stdout to contain none of:"
		for _, thing := range test.StdoutContainsNone {
			str += fmt.Sprintf("\n      - '%s'", thing)
		}
		return str
	}
	return ""
}

func pointerToBool(a bool) *bool {
	return &a
}

func CommandRun(
	data api.LessonDataCLICommand,
	results []api.CLICommandResult,
) {
	commandRenderer(data, results, nil, false)
}

func CommandSubmission(
	data api.LessonDataCLICommand,
	results []api.CLICommandResult,
	failure *api.StructuredErrCLICommand,
) {
	commandRenderer(data, results, failure, true)
}

func commandRenderer(
	data api.LessonDataCLICommand,
	results []api.CLICommandResult,
	failure *api.StructuredErrCLICommand,
	isSubmit bool,
) {
	var wg sync.WaitGroup
	ch := make(chan tea.Msg, 1)
	p := tea.NewProgram(initialModelCmd(isSubmit), tea.WithoutSignalHandler())
	wg.Add(1)
	go func() {
		defer wg.Done()
		if model, err := p.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if r, ok := model.(cmdRootModel); ok {
			r.clear = false
			r.finalized = true
			output := termenv.NewOutput(os.Stdout)
			output.WriteString(r.View())
		}
	}()
	go func() {
		for {
			msg := <-ch
			p.Send(msg)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i, cmd := range data.CLICommandData.Commands {
			ch <- startCmdMsg{cmd: results[i].FinalCommand}
			for _, test := range cmd.Tests {
				ch <- startTestMsg{text: prettyPrintCmd(test)}
			}
			time.Sleep(500 * time.Millisecond)
			earlierCmdFailed := false
			if failure != nil {
				earlierCmdFailed = failure.FailedCommandIndex < i
			}
			for j := range cmd.Tests {
				earlierTestFailed := false
				if failure != nil {
					if earlierCmdFailed {
						earlierTestFailed = true
					} else if failure.FailedCommandIndex == i {
						earlierTestFailed = failure.FailedTestIndex < j
					}
				}
				if !isSubmit {
					ch <- resolveTestMsg{index: j}
				} else if earlierTestFailed {
					ch <- resolveTestMsg{index: j}
				} else {
					time.Sleep(350 * time.Millisecond)
					passed := failure == nil || failure.FailedCommandIndex != i || failure.FailedTestIndex != j
					ch <- resolveTestMsg{index: j, passed: pointerToBool(passed)}
				}
			}
			if !isSubmit {
				ch <- resolveCmdMsg{index: i, results: &results[i]}

			} else if earlierCmdFailed {
				ch <- resolveCmdMsg{index: i}
			} else {
				passed := failure == nil || failure.FailedCommandIndex != i
				if passed {
					ch <- resolveCmdMsg{index: i, passed: pointerToBool(passed)}
				} else {
					ch <- resolveCmdMsg{index: i, passed: pointerToBool(passed), results: &results[i]}
				}
			}
		}
		time.Sleep(500 * time.Millisecond)

		ch <- doneCmdMsg{failure: failure}
	}()
	wg.Wait()
}
