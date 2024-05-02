package cmd

import (
	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/spf13/cobra"
)

var runBaseURL string

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&runBaseURL, "baseurl", "b", "", "set the base URL for HTTP tests, overriding any default")
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:    "run UUID",
	Args:   cobra.ExactArgs(1),
	Short:  "Run an lesson without submitting",
	PreRun: compose(requireUpdated, requireAuth),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		lessonUUID := args[0]
		lesson, err := api.FetchLesson(lessonUUID)
		if err != nil {
			return err
		}
		if lesson.Lesson.Type == "type_http_tests" {
			results, finalBaseURL := checks.HttpTest(*lesson, &runBaseURL)
			printResults(results, lesson, finalBaseURL)
			cobra.CheckErr(err)
		} else {
			cobra.CheckErr("unsupported this lesson type")
		}
		return nil
	},
}
