package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/render"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

func init() {
	rootCmd.AddCommand(localTestCmd)
	localTestCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false, "show detailed output for every step")
}

var localTestCmd = &cobra.Command{
	Use:    "local-test PATH",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	RunE:   localTestHandler,
}

func localTestHandler(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	data, err := readLocalCLIData(args[0])
	if err != nil {
		return err
	}
	if err := validateAllowedOS(data); err != nil {
		return err
	}

	overrideBaseURL := viper.GetString("override_base_url")
	if overrideBaseURL != "" {
		fmt.Printf("Using overridden base_url: %v\n", overrideBaseURL)
		fmt.Printf("You can reset to the default with `bootdev config base_url --reset`\n\n")
	}

	ch := make(chan tea.Msg, 1)
	finalise := render.StartRenderer(data, true, verboseOutput, ch)

	cliResults := checks.CLIChecks(data, overrideBaseURL, ch)
	submissionEvent := checks.LocalSubmissionEvent(data, cliResults)
	checks.ApplySubmissionResults(data, submissionEvent.StructuredErrCLI, ch)
	finalise(submissionEvent)

	if submissionEvent.ResultSlug != api.VerificationResultSlugSuccess {
		return localTestFailureError(submissionEvent.StructuredErrCLI)
	}

	return nil
}

func localTestFailureError(failure *api.StructuredErrCLI) error {
	if failure == nil {
		return errors.New("local checks failed")
	}
	return fmt.Errorf(
		"local checks failed: step %d, test %d\n%s",
		failure.FailedStepIndex+1,
		failure.FailedTestIndex+1,
		failure.ErrorMessage,
	)
}

func readLocalCLIData(path string) (api.CLIData, error) {
	cleanPath := filepath.Clean(path)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return api.CLIData{}, err
	}
	if info.IsDir() {
		cleanPath = filepath.Join(cleanPath, "cli.yaml")
	}

	bytes, err := os.ReadFile(cleanPath)
	if err != nil {
		return api.CLIData{}, err
	}

	var data api.CLIData
	if err := yaml.Unmarshal(bytes, &data); err != nil {
		return api.CLIData{}, err
	}
	if len(data.Steps) == 0 {
		return api.CLIData{}, errors.New("test manifest should include at least one step")
	}

	return data, nil
}

func validateAllowedOS(data api.CLIData) error {
	if len(data.AllowedOperatingSystems) == 0 {
		return nil
	}

	if slices.Contains(data.AllowedOperatingSystems, runtime.GOOS) {
		return nil
	}

	return fmt.Errorf(
		"lesson is not supported for your operating system (%s)\ntry again with one of the following: %v",
		runtime.GOOS,
		data.AllowedOperatingSystems,
	)
}
