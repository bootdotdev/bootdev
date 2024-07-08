package cmd

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "bootdev",
	Short: "The official boot.dev CLI",
	Long: `The official CLI for boot.dev. This program is meant
to be a companion app (not a replacement) for the website.`,
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.bootdev.yaml)")
}

func readViperConfig(paths []string) error {
	for _, path := range paths {
		_, err := os.Stat(path)
		if err == nil {
			viper.SetConfigFile(path)
			break
		}
	}
	return viper.ReadInConfig()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetDefault("base_url", "https://boot.dev")
	viper.SetDefault("api_url", "https://api.boot.dev")
	viper.SetDefault("access_token", "")
	viper.SetDefault("refresh_token", "")
	viper.SetDefault("last_refresh", 0)
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		err := viper.ReadInConfig()
		cobra.CheckErr(err)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// viper's built in config path thing sucks, let's do it ourselves
		defaultPath := path.Join(home, ".bootdev.yaml")
		configPaths := []string{}
		configPaths = append(configPaths, path.Join(home, ".config", "bootdev", "config.yaml"))
		configPaths = append(configPaths, defaultPath)
		if err := readViperConfig(configPaths); err != nil {
			viper.SafeWriteConfigAs(defaultPath)
			viper.SetConfigFile(defaultPath)
			err = viper.ReadInConfig()
			cobra.CheckErr(err)
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
	if info == nil || info.FailedToFetch != nil {
		fmt.Fprintln(os.Stderr, "Failed to fetch update info. Are you online?")
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
