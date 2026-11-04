package render

import (
	"fmt"
	"os"
	"strings"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var (
	passStyle        lipgloss.Style
	failStyle        lipgloss.Style
	accentStyle      lipgloss.Style
	warningStyle     lipgloss.Style
	secondaryStyle   lipgloss.Style
	headerStyle      lipgloss.Style
	subheaderStyle   lipgloss.Style
	informationStyle lipgloss.Style
)

func (m rootModel) Init() tea.Cmd {
	passStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.pass")))
	failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.fail")))
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.accent")))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.warning")))
	secondaryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.secondary")))
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.header")))
	subheaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.subheader")))
	informationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.information")))
	return m.spinner.Tick
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		return m, nil

	case messages.DoneStepMsg:
		m.result = msg.Result
		m.failure = msg.Failure
		m.xpReward = msg.XPReward
		m.xpBreakdown = msg.XPBreakdown
		m.clear = true
		return m, tea.Quit

	case messages.StartStepMsg:
		description := strings.TrimSpace(msg.Description)
		detail := fmt.Sprintf("Command: %s", msg.CMD)
		if msg.TmdlQuery != nil {
			detail += fmt.Sprintf(" (TMDL query: '%s')", *msg.TmdlQuery)
		}
		if msg.CMD == "" {
			detail = fmt.Sprintf("Request: %s %s", msg.Method, msg.URL)
		}
		if description == "" {
			description = strings.TrimPrefix(detail, "Command: ")
			description = strings.TrimPrefix(description, "Request: ")
		}
		m.steps = append(m.steps, stepModel{
			description:     description,
			detail:          detail,
			tests:           []testModel{},
			noPenaltyOnFail: msg.NoPenaltyOnFail,
		})
		return m, nil

	case messages.SleepMsg:
		if len(m.steps) > 0 {
			lastStepIdx := len(m.steps) - 1
			durationSec := float64(msg.DurationMs) / 1000.0
			sleepText := ""
			if durationSec >= 1.0 {
				sleepText = fmt.Sprintf("Waiting %.1fs...", durationSec)
			} else {
				sleepText = fmt.Sprintf("Waiting %dms...", msg.DurationMs)
			}
			m.steps[lastStepIdx].sleepAfter = sleepText
		}
		return m, nil

	case messages.ResolveStepMsg:
		m.steps[msg.Index].passed = msg.Passed
		m.steps[msg.Index].finished = true
		if msg.Result != nil {
			m.steps[msg.Index].result = msg.Result
		}
		return m, nil

	case messages.StartTestMsg:
		m.steps[len(m.steps)-1].tests = append(
			m.steps[len(m.steps)-1].tests,
			testModel{text: msg.Text},
		)
		return m, nil

	case messages.ResolveTestMsg:
		m.steps[msg.StepIndex].tests[msg.TestIndex].passed = msg.Passed
		m.steps[msg.StepIndex].tests[msg.TestIndex].finished = true
		return m, nil

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func StartRenderer(isSubmit bool, verbose bool, showOmitLessonIDTip bool, runContext RunContext) (func(tea.Msg), func(api.LessonSubmissionEvent)) {
	m := initModel(isSubmit, verbose)
	if terminalWidth, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		m.terminalWidth = terminalWidth
	}
	m.runContext = runContext
	m.showOmitLessonIDTip = showOmitLessonIDTip
	interactive := term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	options := []tea.ProgramOption{tea.WithoutSignalHandler(), tea.WithInput(nil)}
	if !interactive {
		options = append(options, tea.WithoutRenderer())
	}
	p := tea.NewProgram(m, options...)
	done := make(chan struct{})

	go func() {
		defer close(done)
		if model, err := p.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if r, ok := model.(rootModel); ok {
			r.clear = false
			r.finalized = true
			output := r.View()
			if !interactive {
				output = ansi.Strip(output)
			}
			fmt.Fprint(os.Stdout, output)
		}
	}()

	finish := func(submissionEvent api.LessonSubmissionEvent) {
		p.Send(messages.DoneStepMsg{
			Result:      submissionEvent.ResultSlug,
			Failure:     submissionEvent.StructuredErrCLI,
			XPReward:    submissionEvent.XPReward,
			XPBreakdown: submissionEvent.XPBreakdown,
		})
		<-done
	}

	return p.Send, finish
}
