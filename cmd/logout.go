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
	r, err := http.NewRequest("POST", api_url+"/v1/auth/logout", bytes.NewBuffer([]byte{}))
	r.Header.Add("X-Refresh-Token", viper.GetString("refresh_token"))
	client.Do(r)

	cobra.CheckErr(err)

	viper.Set("access_token", "")
	viper.Set("refresh_token", "")
	viper.Set("last_refresh", time.Now().Unix())
	viper.WriteConfig()
	fmt.Println("Logged out successfully.")
}

var logoutCmd = &cobra.Command{
	Use:     "logout",
	Aliases: []string{"signout"},
	Short:   "Disconnect the CLI from your account",
	Run: func(cmd *cobra.Command, args []string) {
		requireAuth()
		logout()
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
