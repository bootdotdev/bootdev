package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var loginCmd = &cobra.Command{
	Use:          "login",
	Aliases:      []string{"auth", "authenticate", "signin"},
	Short:        "Authenticate the CLI with your account",
	SilenceUsage: true,
	PreRun:       requireUpdated,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: check if the logo fits screen width
		// fmt.Print(logo)
		fmt.Print("Welcome to the boot.dev CLI!\n\n")

		fmt.Println("Please navigate to:\n" +
			viper.GetString("base_url") +
			"/cli/login?redirect=/cli/login")

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nPaste your login code: ")
		text, err := reader.ReadString('\n')

		if err != nil {
			return err
		}

		re := regexp.MustCompile(`[^A-Za-z0-9_-]`)
		text = re.ReplaceAllString(text, "")
		creds, err := api.LoginWithCode(text)
		if err != nil {
			return err
		}

		if creds.AccessToken == "" || creds.RefreshToken == "" {
			return errors.New("invalid credentials received")
		}

		viper.Set("access_token", creds.AccessToken)
		viper.Set("refresh_token", creds.RefreshToken)
		viper.Set("last_refresh", time.Now().Unix())

		err = viper.WriteConfig()
		if err != nil {
			return err
		}

		fmt.Println("Logged in successfully!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
