package cmd

import (
	"fmt"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication and CLI version status",
	Long:  "Display whether you're logged in and whether the CLI is up to date",
	Run: func(cmd *cobra.Command, args []string) {
		checkAuthStatus()
		fmt.Println() // Blank line for readability
		checkVersionStatus(cmd)
	},
}

func checkAuthStatus() {
	refreshToken := viper.GetString("refresh_token")
	if refreshToken == "" {
		fmt.Println("Not logged in")
		fmt.Println("Run 'bootdev login' to authenticate")
		return
	}

	// Verify token is still valid by attempting to refresh
	_, err := api.FetchAccessToken()
	if err != nil {
		fmt.Println("Authentication expired")
		fmt.Println("Run 'bootdev login' to re-authenticate")
		return
	}

	fmt.Println("Logged in")
	// TODO: Consider adding user data endpoint to show email/username
}

func checkVersionStatus(cmd *cobra.Command) {
	info := version.FromContext(cmd.Context())
	if info == nil || info.FailedToFetch != nil {
		fmt.Println("Unable to check version status")
		if info != nil && info.FailedToFetch != nil {
			fmt.Printf("Error: %s\n", info.FailedToFetch.Error())
		}
		return
	}

	if info.IsOutdated {
		fmt.Printf("CLI outdated: %s → %s available\n", info.CurrentVersion, info.LatestVersion)
		fmt.Println("Run 'bootdev upgrade' to update")
	} else {
		fmt.Printf("CLI up to date (%s)\n", info.CurrentVersion)
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
