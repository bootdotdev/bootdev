package render

import (
	api "github.com/bootdotdev/bootdev/client"
	"github.com/charmbracelet/bubbles/spinner"
)

type testModel struct {
	text     string
	passed   *bool
	finished bool
}

type stepModel struct {
	description     string
	detail          string
	passed          *bool
	result          *api.CLIStepResult
	finished        bool
	tests           []testModel
	sleepAfter      string
	noPenaltyOnFail bool
}

type rootModel struct {
	steps       []stepModel
	spinner     spinner.Model
	result      api.VerificationResultSlug
	failure     *api.StructuredErrCLI
	xpReward    int
	xpBreakdown []api.XPBreakdownItem
	isSubmit    bool
	verbose     bool
	finalized   bool
	clear       bool
}

func initModel(isSubmit bool, verbose bool) rootModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return rootModel{
		spinner:  s,
		isSubmit: isSubmit,
		verbose:  verbose,
		steps:    []stepModel{},
	}
}
