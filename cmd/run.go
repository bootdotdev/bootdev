package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&submitBaseURL, "baseurl", "b", "", "set the base URL for HTTP tests, overriding any default")
	runCmd.Flags().BoolVarP(&forceSubmit, "submit", "s", false, "shortcut flag to submit instead of run")
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:    "run UUID",
	Args:   cobra.MatchAll(cobra.RangeArgs(1, 10)),
	Short:  "Run a lesson without submitting",
	PreRun: compose(requireUpdated, requireAuth),
	RunE:   submissionHandler,
}
