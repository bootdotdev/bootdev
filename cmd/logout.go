package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func logout() {
	api_url := viper.GetString("api_url")
	client := &http.Client{}
	// Best effort - logout should never fail
	r, _ := http.NewRequest("POST", api_url+"/v1/auth/logout", bytes.NewBuffer([]byte{}))
	r.Header.Add("X-Refresh-Token", viper.GetString("refresh_token"))
	client.Do(r)

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
