package cmd

import (
	"fmt"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func logout() error {
	// Server logout is best effort. Local credentials should still be cleared
	// when the token is expired or the API is unavailable.
	if viper.GetString("refresh_token") != "" {
		_ = api.Logout()
	}

	viper.Set("access_token", "")
	viper.Set("refresh_token", "")
	viper.Set("last_refresh", 0)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to clear stored credentials: %w", err)
	}

	fmt.Println("Logged out successfully.")
	return nil
}

var logoutCmd = &cobra.Command{
	Use:          "logout",
	Aliases:      []string{"signout"},
	Short:        "Disconnect the CLI from your account",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return logout()
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
