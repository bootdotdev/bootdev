package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/spf13/cobra"
)

var httpTestPort int

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:    "submit UUID",
	Args:   cobra.MatchAll(cobra.ExactArgs(1)),
	Short:  "Submit an assignment",
	PreRun: requireAuth,
	Run: func(cmd *cobra.Command, args []string) {
		assignmentUUID := args[0]
		assignment, err := api.FetchAssignment(assignmentUUID)
		cobra.CheckErr(err)
		if assignment.Assignment.Type == "type_http_tests" {
			results := checks.HttpTest(*assignment, httpTestPort)
			cobra.CheckErr(err)
			submitResults, err := api.SubmitHTTPTestAssignment(assignmentUUID, results)
			cobra.CheckErr(err)

			// TODO: parse these results
			bytes, _ := json.Marshal(submitResults)
			fmt.Println(string(bytes))
		}
	},
}

func init() {
	rootCmd.AddCommand(submitCmd)
	submitCmd.Flags().IntVarP(&httpTestPort, "port", "p", 8080, "port for HTTP tests")
}
