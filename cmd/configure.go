package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configColors []*string

var defaultColorKeys []string

var defaultColors = map[string]string{
	"gray":  "8",
	"red":   "1",
	"green": "2",
}

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Change configuration of the CLI",
	Run: func(cmd *cobra.Command, args []string) {
		showHelp := true
		for i, color := range configColors {
			key := "color." + defaultColorKeys[i]
			if color != nil {
				if *color == "" {
					viper.Set(key, defaultColors[defaultColorKeys[i]])
					fmt.Println("unset " + key)
				} else {
					if viper.GetString(key) == *color {
						continue
					}
					viper.Set(key, *color)
					style := lipgloss.NewStyle().Foreground(lipgloss.Color(*color))
					fmt.Println("set " + style.Render(key) + "!")
				}
				showHelp = false
			}
		}
		if showHelp {
			cmd.Help()
		} else {
			viper.WriteConfig()
		}

	},
}

func init() {
	rootCmd.AddCommand(configureCmd)

	configColors = make([]*string, len(defaultColors))
	defaultColorKeys = make([]string, len(defaultColors))
	for color, def := range defaultColors {
		configColors = append(configColors, configureCmd.Flags().String("color-"+color, def, "ANSI number or hex string"))
		defaultColorKeys = append(defaultColorKeys, color)
		viper.SetDefault("color."+color, def)
	}
}
