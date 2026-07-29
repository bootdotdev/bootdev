package cmd

import (
	"fmt"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func logout() {
	// Best effort - logout should never fail
	_ = api.Logout()

	viper.Set("access_token", "")
	viper.Set("refresh_token", "")
	viper.Set("last_refresh", time.Now().Unix())
	viper.WriteConfig()
	fmt.Println("Logged out successfully.")
}

var logoutCmd = &cobra.Command{
	Use:          "logout",
	Aliases:      []string{"signout"},
	Short:        "Disconnect the CLI from your account",
	PreRun:       requireAuth,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		logout()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
