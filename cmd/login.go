package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const logo string = `
			 @@@@                                                             @@@@
		@@@@@@@@@@@ @@@@@@@                 @@@@                @@@@@@@ @@@@@@@@@@@
	 @@@      @@@@   @@@@@@@@@@@@@@@@@@@@@@  @@@@@@@@@@@@@@@@@@@@@   @@@@     @@@@
	@@@                                       ...                          .. . @@@
 @@@         @@@@@@@                           @@@@@@@@                    .   @@@
@@@   .       @@   @@  @@@@   @@@@  @@@@@@@@    @@    @@ @@@@@@ @@@   @@@       @@@
@@@  ..       @@@@@@  @@  @@ @@  @@ @  @@  @    @@     @@ @@     @@  .@@        @@@@
 @@@  ..      @@   @@ @@  @@ @@  @@    @@       @@     @@ @@@@    @@ @@        @@@@
	@@@   .     @@   @@ @@  @@ @@  @@    @@       @@    @@  @@       @@@        @@@
	 @@@       @@@@@@@   @@@@   @@@@     @@   @@ @@@@@@@@  @@@@@@     @    ..  @@@
		@@@             .                                                     ..@@@
		 @@@@   @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@   @@@
			 @@@@@@                                                          @@@@@@
				 @																																@
`

type LoginRequest struct {
	Otp string `json:"otp"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func loginWithCode(code string) {
	api_url := viper.GetString("api_url")
	req, err := json.Marshal(LoginRequest{Otp: code})
	cobra.CheckErr(err)

	resp, err := http.Post(api_url+"/v1/auth/otp/login", "application/json", bytes.NewReader(req))
	cobra.CheckErr(err)

	if resp.StatusCode != 200 {
		cobra.CheckErr(errors.New("Invalid login code"))
	}

	body, err := io.ReadAll(resp.Body)
	cobra.CheckErr(err)

	var creds LoginResponse
	err = json.Unmarshal(body, &creds)
	cobra.CheckErr(err)
	if creds.AccessToken == "" || creds.RefreshToken == "" {
		cobra.CheckErr(errors.New("Invalid credentials received"))
	}
	viper.Set("access_token", creds.AccessToken)
	viper.Set("refresh_token", creds.RefreshToken)
	viper.Set("last_refresh", time.Now().Unix())
	viper.WriteConfig()
	// TODO: check if the logo fits
	fmt.Print(logo)
	fmt.Println("Logged in successfully!")
}

var loginCmd = &cobra.Command{
	Use:     "login",
	Aliases: []string{"auth", "authenticate", "signin"},
	Short:   "Authenticate the CLI with your account",
	Run: func(cmd *cobra.Command, args []string) {
		baseUrl := viper.GetString("base_url")
		fmt.Print("Welcome to the boot.dev CLI!\n\n")
		fmt.Println("Please navigate to:\n" + baseUrl + "/cli/login?redirect=/cli/login")

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nPaste your login code: ")
		text, err := reader.ReadString('\n')
		cobra.CheckErr(err)

		loginWithCode(strings.Trim(text, " \n"))
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
