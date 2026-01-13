package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "bootdev",
	Short: "Official Boot.dev CLI",
	Long: `The official CLI for Boot.dev. This program is meant
as a companion app (not a replacement) for the website.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(currentVersion string) error {
	rootCmd.Version = currentVersion
	info := version.FetchUpdateInfo(rootCmd.Version)
	defer info.PromptUpdateIfAvailable()
	ctx := version.WithContext(context.Background(), &info)
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.bootdev.yaml or $XDG_CONFIG_HOME/bootdev/config.yaml)")
}

func readViperConfig(paths []string) error {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			viper.SetConfigFile(p)
			break
		}
	}
	return viper.ReadInConfig()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetDefault("frontend_url", "https://boot.dev")
	viper.SetDefault("api_url", "https://api.boot.dev")
	viper.SetDefault("access_token", "")
	viper.SetDefault("refresh_token", "")
	viper.SetDefault("last_refresh", 0)
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(filepath.Clean(cfgFile))
		cobra.CheckErr(viper.ReadInConfig())
	} else {
		// find home dir
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// collect paths where existing config files may be located
		var configPaths []string

		// first check XDG_CONFIG_HOME if set
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		var xdgEnvPath string
		if xdgConfigHome != "" {
			xdgEnvPath = filepath.Join(xdgConfigHome, "bootdev", "config.yaml")
			configPaths = append(configPaths, xdgEnvPath)
		}

		// then check legacy hard-coded "XDG" path, then home dotfile
		xdgLegacyPath := filepath.Join(home, ".config", "bootdev", "config.yaml")
		homeDotfilePath := filepath.Join(home, ".bootdev.yaml")

		configPaths = append(configPaths, xdgLegacyPath)
		configPaths = append(configPaths, homeDotfilePath)

		if err := readViperConfig(configPaths); err != nil {
			// no existing config found; try to create a new one
			// respect XDG_CONFIG_HOME if set, otherwise use dotfile in home dir
			var newConfigPath string
			if xdgEnvPath != "" {
				newConfigPath = xdgEnvPath
				cobra.CheckErr(os.MkdirAll(filepath.Dir(newConfigPath), 0o755))
			} else {
				newConfigPath = homeDotfilePath
			}

			cobra.CheckErr(viper.SafeWriteConfigAs(newConfigPath))
			viper.SetConfigFile(newConfigPath)
			cobra.CheckErr(viper.ReadInConfig())
		}
	}

	viper.SetEnvPrefix("bd")
	viper.AutomaticEnv() // read in environment variables that match
}

// Chain multiple commands together.
func compose(commands ...func(cmd *cobra.Command, args []string)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		for _, command := range commands {
			command(cmd, args)
		}
	}
}

// Call this function at the beginning of a command handler
// if you want to require the user to update their CLI first.
func requireUpdated(cmd *cobra.Command, args []string) {
	info := version.FromContext(cmd.Context())
	if info == nil {
		fmt.Fprintln(os.Stderr, "Failed to fetch update info. Are you online?")
		os.Exit(1)
	}
	if info.FailedToFetch != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch update info: %s\n", info.FailedToFetch.Error())
		os.Exit(1)
	}
	if info.IsUpdateRequired {
		info.PromptUpdateIfAvailable()
		os.Exit(1)
	}
}

// Call this function at the beginning of a command handler
// if you need to make authenticated requests. This will
// automatically refresh the tokens, if necessary, and prompt
// the user to re-login if anything goes wrong.
func requireAuth(cmd *cobra.Command, args []string) {
	promptLoginAndExitIf := func(condition bool) {
		if condition {
			fmt.Fprintln(os.Stderr, "You must be logged in to use that command.")
			fmt.Fprintln(os.Stderr, "Please run 'bootdev login' first.")
			os.Exit(1)
		}
	}

	access_token := viper.GetString("access_token")
	promptLoginAndExitIf(access_token == "")

	// We only refresh if our token is getting stale.
	last_refresh := viper.GetInt64("last_refresh")
	if time.Now().Add(-time.Minute*55).Unix() <= last_refresh {
		return
	}

	creds, err := api.FetchAccessToken()
	promptLoginAndExitIf(err != nil)
	if creds.AccessToken == "" || creds.RefreshToken == "" {
		promptLoginAndExitIf(err != nil)
	}

	viper.Set("access_token", creds.AccessToken)
	viper.Set("refresh_token", creds.RefreshToken)
	viper.Set("last_refresh", time.Now().Unix())

	err = viper.WriteConfig()
	promptLoginAndExitIf(err != nil)
}
