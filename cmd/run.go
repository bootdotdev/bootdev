package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&submitBaseURL, "baseurl", "b", "", "set the base URL for HTTP tests, overriding any default")
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:    "run UUID",
	Args:   cobra.MatchAll(cobra.RangeArgs(1, 10)),
	Short:  "Run an assignment without submitting",
	PreRun: compose(requireUpdated, requireAuth),
	RunE:   submissionHandler,
}
