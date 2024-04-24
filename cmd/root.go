package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

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
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.bootdev.yaml)")
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

		// Search config in home directory with name ".bootdev" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".bootdev")
		if err := viper.ReadInConfig(); err != nil {
			home, err := os.UserHomeDir()
			cobra.CheckErr(err)
			viper.SafeWriteConfigAs(path.Join(home, ".bootdev.yaml"))
			viper.ReadInConfig()
			cobra.CheckErr(err)
		}
	}

	viper.SetEnvPrefix("bd")
	viper.AutomaticEnv() // read in environment variables that match
}

func promptLoginAndExitIf(condition bool) {
	if condition {
		fmt.Println("You must be logged in to use that command.")
		fmt.Println("Please run 'bootdev login' first.")
		os.Exit(1)
	}
}

// Call this function at the beginning of a command handler
// if you need to make authenticated requests. This will
// automatically refresh the tokens, if necessary, and prompt
// the user to re-login if anything goes wrong.
func requireAuth() {
	access_token := viper.GetString("access_token")
	promptLoginAndExitIf(access_token == "")

	// We only refresh if our token is getting stale.
	last_refresh := viper.GetInt64("last_refresh")
	if time.Now().Add(-time.Minute*55).Unix() <= last_refresh {
		return
	}

	api_url := viper.GetString("api_url")

	client := &http.Client{}
	r, err := http.NewRequest("POST", api_url+"/v1/auth/refresh", bytes.NewBuffer([]byte{}))
	r.Header.Add("X-Refresh-Token", viper.GetString("refresh_token"))
	promptLoginAndExitIf(err != nil)
	resp, err := client.Do(r)
	promptLoginAndExitIf(err != nil)

	defer resp.Body.Close()
	promptLoginAndExitIf(err != nil)

	if resp.StatusCode != 200 {
		promptLoginAndExitIf(err != nil)
	}

	body, err := io.ReadAll(resp.Body)
	promptLoginAndExitIf(err != nil)

	var creds LoginResponse
	err = json.Unmarshal(body, &creds)
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
