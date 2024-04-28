package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var loginCmd = &cobra.Command{
	Use:     "login",
	Aliases: []string{"auth", "authenticate", "signin"},
	Short:   "Authenticate the CLI with your account",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: check if the logo fits screen width
		// fmt.Print(logo)
		fmt.Print("Welcome to the boot.dev CLI!\n\n")

		fmt.Println("Please navigate to:\n" +
			viper.GetString("base_url") +
			"/cli/login?redirect=/cli/login")

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nPaste your login code: ")
		text, err := reader.ReadString('\n')
		cobra.CheckErr(err)

		creds, err := api.LoginWithCode(strings.Trim(text, " \n"))
		cobra.CheckErr(err)
		if creds.AccessToken == "" || creds.RefreshToken == "" {
			cobra.CheckErr(errors.New("invalid credentials received"))
		}

		viper.Set("access_token", creds.AccessToken)
		viper.Set("refresh_token", creds.RefreshToken)
		viper.Set("last_refresh", time.Now().Unix())

		err = viper.WriteConfig()
		cobra.CheckErr(err)
		fmt.Println("Logged in successfully!")
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
