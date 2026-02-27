package render

import (
	"fmt"
	"os"
	"sync"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/viper"
)

var (
	green     lipgloss.Style
	red       lipgloss.Style
	gray      lipgloss.Style
	borderBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
)

func (m rootModel) Init() tea.Cmd {
	green = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.green")))
	red = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.red")))
	gray = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.gray")))
	return m.spinner.Tick
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.DoneStepMsg:
		m.result = msg.Result
		m.failure = msg.Failure
		m.clear = true
		return m, tea.Quit

	case messages.StartStepMsg:
		step := fmt.Sprintf("Running: %s", msg.CMD)
		if msg.TmdlQuery != nil {
			step += fmt.Sprintf(" (TMDL query: '%s')", *msg.TmdlQuery)
		}
		if msg.CMD == "" {
			step = fmt.Sprintf("%s %s", msg.Method, msg.URL)
		}
		m.steps = append(m.steps, stepModel{
			step:              step,
			tests:             []testModel{},
			responseVariables: msg.ResponseVariables,
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

func StartRenderer(data api.CLIData, isSubmit bool, ch chan tea.Msg) func(api.LessonSubmissionEvent) {
	var wg sync.WaitGroup
	p := tea.NewProgram(initModel(isSubmit), tea.WithoutSignalHandler())

	wg.Add(1)
	go func() {
		defer wg.Done()
		if model, err := p.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if r, ok := model.(rootModel); ok {
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

	return func(submissionEvent api.LessonSubmissionEvent) {
		ch <- messages.DoneStepMsg{
			Result:  submissionEvent.ResultSlug,
			Failure: submissionEvent.StructuredErrCLI,
		}
		wg.Wait()
	}
}
