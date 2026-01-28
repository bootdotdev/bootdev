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
	responseVariables []api.HTTPRequestResponseVariable
	step              string
	passed            *bool
	result            *api.CLIStepResult
	finished          bool
	tests             []testModel
	sleepAfter        string
}

type rootModel struct {
	steps     []stepModel
	spinner   spinner.Model
	failure   *api.VerificationResultStructuredErrCLI
	isSubmit  bool
	success   bool
	finalized bool
	clear     bool
}

func initModel(isSubmit bool) rootModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return rootModel{
		spinner:  s,
		isSubmit: isSubmit,
		steps:    []stepModel{},
	}
}
