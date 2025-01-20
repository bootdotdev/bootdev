package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var resetAll bool
var resetSubmitBaseURL bool
var resetColors bool

func init() {
	rootCmd.AddCommand(resetCmd)
	resetCmd.Flags().BoolVarP(&resetAll, "all", "a", false, "rest all config (not including login data)")
	resetCmd.Flags().BoolVarP(&resetSubmitBaseURL, "base_url", "b", false, "reset base URL to use the lesson's default")
	resetCmd.Flags().BoolVarP(&resetColors, "colors", "c", false, "reset colors")
}

// resetCmd represents the reset command
var resetCmd = &cobra.Command{
	Use:    "reset",
	Short:  "Reset config options to their default values",
	PreRun: compose(requireUpdated, requireAuth),
	Run: func(cmd *cobra.Command, args []string) {
		showHelp := true

		if resetAll {
			resetSubmitBaseURL = true
			resetColors = true
		}

		if resetSubmitBaseURL {
			viper.Set("submit_base_url", "")
			showHelp = false
		}

		if resetColors {
			for color, value := range defaultColors {
				key := "color." + color
				viper.Set(key, value)
			}
			showHelp = false
		}

		if showHelp {
			cmd.Help()
		} else {
			viper.WriteConfig()
		}
	},
}
