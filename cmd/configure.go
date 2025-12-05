package cmd

import (
	"fmt"
	"net/url"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configureCmd represents the configure command which is a container for other
// sub-commands (e.g., colors, base URL override)
var configureCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"configure"},
	Short:   "Change configuration of the CLI",
}

var defaultColors = map[string]string{
	"gray":  "8",
	"red":   "1",
	"green": "2",
}

// configureColorsCmd represents the `configure colors` command for changing
// the colors of the text output
var configureColorsCmd = &cobra.Command{
	Use:   "colors",
	Short: "Get or set the CLI text colors",
	RunE: func(cmd *cobra.Command, args []string) error {
		resetColors, err := cmd.Flags().GetBool("reset")
		if err != nil {
			return fmt.Errorf("couldn't get the reset flag value: %v", err)
		}

		if resetColors {
			for color, defaultVal := range defaultColors {
				viper.Set("color."+color, defaultVal)
			}

			err := viper.WriteConfig()
			if err != nil {
				return fmt.Errorf("failed to write config: %v", err)
			}

			fmt.Println("Colors reset!")
			return err
		}

		configColors := map[string]string{}
		for color := range defaultColors {
			configVal, err := cmd.Flags().GetString(color)
			if err != nil {
				return fmt.Errorf("couldn't get the %v flag value: %v", color, err)
			}

			configColors[color] = configVal
		}

		noFlags := true
		for color, configVal := range configColors {
			if configVal == "" {
				continue
			}

			noFlags = false
			key := "color." + color
			viper.Set(key, configVal)
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(configVal))
			fmt.Println("set " + style.Render(key) + "!")
		}

		if noFlags {
			for color := range configColors {
				val := viper.GetString("color." + color)
				style := lipgloss.NewStyle().Foreground(lipgloss.Color(val))
				fmt.Printf(style.Render("%v: %v")+"\n", color, val)
			}
			return nil
		}

		err = viper.WriteConfig()
		if err != nil {
			return fmt.Errorf("failed to write config: %v", err)
		}
		return err
	},
}

// configureBaseURLCmd represents the `configure base_url` command
var configureBaseURLCmd = &cobra.Command{
	Use:   "base_url [url]",
	Short: "Get or set the base URL for HTTP tests, overriding lesson defaults",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resetOverrideBaseURL, err := cmd.Flags().GetBool("reset")
		if err != nil {
			return fmt.Errorf("couldn't get the reset flag value: %v", err)
		}

		if resetOverrideBaseURL {
			viper.Set("override_base_url", "")
			err := viper.WriteConfig()
			if err != nil {
				return fmt.Errorf("failed to write config: %v", err)
			}
			fmt.Println("Base URL reset!")
			return err
		}

		if len(args) == 0 {
			baseURL := viper.GetString("override_base_url")
			message := fmt.Sprintf("Base URL: %s", baseURL)
			if baseURL == "" {
				message = "No base URL set"
			}
			fmt.Println(message)
			return nil
		}

		overrideBaseURL, err := url.Parse(args[0])
		if err != nil {
			return fmt.Errorf("failed to parse base URL: %v", err)
		}
		// for urls like "localhost:8080" the parser reads "localhost" into
		// `Scheme` and leaves `Host` as an empty string, so we must check for
		// both
		if overrideBaseURL.Scheme == "" || overrideBaseURL.Host == "" {
			return fmt.Errorf("invalid URL: provide both protocol scheme and hostname")
		}
		if overrideBaseURL.Scheme == "https" {
			fmt.Println("warning: protocol scheme is set to https")
		}

		viper.Set("override_base_url", overrideBaseURL.String())
		err = viper.WriteConfig()
		if err != nil {
			return fmt.Errorf("failed to write config: %v", err)
		}
		fmt.Printf("Base URL set to %v\n", overrideBaseURL.String())
		return err
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)

	configureCmd.AddCommand(configureBaseURLCmd)
	configureBaseURLCmd.Flags().Bool("reset", false, "reset the base URL to use the lesson's defaults")

	configureCmd.AddCommand(configureColorsCmd)
	configureColorsCmd.Flags().Bool("reset", false, "reset colors to their default values")
	for color, defaultVal := range defaultColors {
		configureColorsCmd.Flags().String(color, "", "ANSI number or hex string")
		viper.SetDefault("color."+color, defaultVal)
	}
}
