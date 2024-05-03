package cmd

import (
	"errors"
	"fmt"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/render"
	"github.com/spf13/cobra"
)

var submitBaseURL string

func init() {
	rootCmd.AddCommand(submitCmd)
	submitCmd.Flags().StringVarP(&submitBaseURL, "baseurl", "b", "", "set the base URL for HTTP tests, overriding any default")
}

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:    "submit UUID",
	Args:   cobra.MatchAll(cobra.ExactArgs(1)),
	Short:  "Submit an assignment",
	PreRun: compose(requireUpdated, requireAuth),
	RunE:   submissionHandler,
}

func submissionHandler(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	isSubmit := cmd.Name() == "submit"
	assignmentUUID := args[0]
	assignment, err := api.FetchAssignment(assignmentUUID)
	if err != nil {
		return err
	}
	switch assignment.Assignment.Type {
	case "type_http_tests":
		results, finalBaseURL := checks.HttpTest(*assignment, &submitBaseURL)
		render.PrintHTTPResults(results, assignment, finalBaseURL)
		if isSubmit {
			err := api.SubmitHTTPTestAssignment(assignmentUUID, results)
			if err != nil {
				return err
			}
			fmt.Println("\nSubmitted! Check the lesson on Boot.dev for results")
		}
	case "type_cli_command":
		results := checks.CLICommand(*assignment)
		data := *assignment.Assignment.AssignmentDataCLICommand
		if isSubmit {
			failure, err := api.SubmitCLICommandAssignment(assignmentUUID, results)
			if err != nil {
				return err
			}
			render.CommandSubmission(data, results, failure)
		} else {
			render.CommandRun(data, results)
		}
	default:
		return errors.New("unsupported assignment type")
	}
	return nil
}
