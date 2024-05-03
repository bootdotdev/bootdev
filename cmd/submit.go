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
	Args:   cobra.MatchAll(cobra.RangeArgs(1, 10)),
	Short:  "Submit an lesson",
	PreRun: compose(requireUpdated, requireAuth),
	RunE:   submissionHandler,
}

func submissionHandler(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	isSubmit := cmd.Name() == "submit"
	lessonUUID := args[0]
	optionalPositionalArgs := []string{}
	if len(args) > 1 {
		optionalPositionalArgs = args[1:]
	}

	lesson, err := api.FetchLesson(lessonUUID)
	if err != nil {
		return err
	}
	switch lesson.Lesson.Type {
	case "type_http_tests":
		results, finalBaseURL := checks.HttpTest(*lesson, &submitBaseURL)
		render.PrintHTTPResults(results, lesson, finalBaseURL)
		if isSubmit {
			err := api.SubmitHTTPTestLesson(lessonUUID, results)
			if err != nil {
				return err
			}
			fmt.Println("\nSubmitted! Check the lesson on Boot.dev for results")
		}
	case "type_cli_command":
		results := checks.CLICommand(*lesson, optionalPositionalArgs)
		data := *lesson.Lesson.LessonDataCLICommand
		if isSubmit {
			failure, err := api.SubmitCLICommandLesson(lessonUUID, results)
			if err != nil {
				return err
			}
			render.CommandSubmission(data, results, failure, optionalPositionalArgs)
		} else {
			render.CommandRun(data, results, optionalPositionalArgs)
		}
	default:
		return errors.New("unsupported lesson type")
	}
	return nil
}
