package cmd

import (
	"errors"
	"fmt"

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
	Args:   cobra.MatchAll(cobra.ExactArgs(1)),
	Short:  "Run an assignment without submitting",
	PreRun: requireAuth,
	Run: func(cmd *cobra.Command, args []string) {
		assignmentUUID := args[0]
		assignment, err := api.FetchAssignment(assignmentUUID)
		cobra.CheckErr(err)
		if assignment.Assignment.Type == "type_http_tests" {
			results := checks.HttpTest(*assignment, &runBaseURL)
			fmt.Println("=====================================")
			defer fmt.Println("=====================================")
			fmt.Println("Running requests:")
			for i, result := range results {
				printResult(result, assignment, i)
			}
			cobra.CheckErr(err)
		} else {
			cobra.CheckErr(errors.New("unsupported assignment type"))
		}
	},
}
