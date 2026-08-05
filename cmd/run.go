package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVarP(&forceSubmit, "submit", "s", false, "shortcut flag to submit after running")
	runCmd.Flags().BoolVar(&debugSubmission, "debug", false, "log submission request/response debug output")
	runCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false, "with --submit, show detailed final output for every step")
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:    "run [UUID]",
	Args:   cobra.MatchAll(cobra.MaximumNArgs(1)),
	Short:  "Run a lesson without submitting. Runs your next lesson when no UUID is given",
	PreRun: compose(requireUpdated, requireAuth),
	RunE:   submissionHandler,
}
