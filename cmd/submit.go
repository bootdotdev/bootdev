package cmd

import (
	"errors"
	"fmt"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/render"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var forceSubmit bool

func init() {
	rootCmd.AddCommand(submitCmd)
}

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:    "submit UUID",
	Args:   cobra.MatchAll(cobra.RangeArgs(1, 10)),
	Short:  "Submit a lesson",
	PreRun: compose(requireUpdated, requireAuth),
	RunE:   submissionHandler,
}

func submissionHandler(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	isSubmit := cmd.Name() == "submit" || forceSubmit
	lessonUUID := args[0]

	lesson, err := api.FetchLesson(lessonUUID)
	if err != nil {
		return err
	}
	if lesson.Lesson.Type != "type_cli" {
		return errors.New("unable to run lesson: unsupported lesson type")
	}
	if lesson.Lesson.LessonDataCLI == nil {
		return errors.New("unable to run lesson: missing lesson data")
	}

	data := lesson.Lesson.LessonDataCLI.CLIData
	overrideBaseURL := viper.GetString("override_base_url")
	if overrideBaseURL != "" {
		fmt.Printf("Using overridden base_url: %v\n", overrideBaseURL)
		fmt.Printf("You can reset to the default with `bootdev config base_url --reset`\n\n")
	}

	results := checks.CLIChecks(data, overrideBaseURL)
	if isSubmit {
		failure, err := api.SubmitCLILesson(lessonUUID, results)
		if err != nil {
			return err
		}
		render.RenderSubmission(data, results, failure)
	} else {
		render.RenderRun(data, results)
	}
	return nil
}
