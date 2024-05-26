package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func logoRenderer() string {
	blue := lipgloss.NewStyle().Foreground(lipgloss.Color("#7e88f7"))
	gray := lipgloss.NewStyle().Foreground(lipgloss.Color("#7e7e81"))
	white := lipgloss.NewStyle().Foreground(lipgloss.Color("#d9d9de"))
	var output string
	var result string
	var prev rune
	for _, c := range logo {
		if c == ' ' {
			result += string(c)
			continue
		}
		if prev != c {
			if len(result) > 0 {
				text := strings.ReplaceAll(result, "B", "@")
				text = strings.ReplaceAll(text, "D", "@")
				switch result[0] {
				case 'B':
					output += white.Render(text)
				case 'D':
					output += blue.Render(text)
				default:
					output += gray.Render(text)
				}
			}
			result = ""
		}
		result += string(c)
		prev = c
	}
	return output
}

const logo string = `
        @@@@                                                           @@@@
    @@@@@@@@@@@ @@@@@@@                 @@@@                @@@@@@@ @@@@@@@@@@@
   @@@      @@@@   @@@@@@@@@@@@@@@@@@@@@@  @@@@@@@@@@@@@@@@@@@@@   @@@@     @@@@
  @@@                                       ...                          .. . @@@
 @@@         BBBBBBB                           DDDDDDDD                    .   @@@
@@@   .       BB   BB  BBBB   BBBB  BBBBBBBB    DD    DD DDDDDD DDD   DDD       @@@
@@@  ..       BBBBBB  BB  BB BB  BB B  BB  B    DD     DD DD     DD  .DD        @@@@
 @@@  ..      BB   BB BB  BB BB  BB    BB       DD     DD DDDD    DD DD        @@@@
  @@@   .     BB   BB BB  BB BB  BB    BB       DD    DD  DD       DDD        @@@
   @@@       BBBBBBB   BBBB   BBBB     BB   BB DDDDDDDD  DDDDDD     D    ..  @@@
    @@@             .                                                     ..@@@
     @@@@   @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@   @@@
      @@@@@@                                                          @@@@@@
          @                                                              @`

var loginCmd = &cobra.Command{
	Use:          "login",
	Aliases:      []string{"auth", "authenticate", "signin"},
	Short:        "Authenticate the CLI with your account",
	SilenceUsage: true,
	PreRun:       requireUpdated,
	RunE: func(cmd *cobra.Command, args []string) error {
		w, _, err := term.GetSize(0)
		if err != nil {
			w = 0
		}
		// Pad the logo with whitespace
		welcome := lipgloss.PlaceHorizontal(lipgloss.Width(logo), lipgloss.Center, "Welcome to the boot.dev CLI!")

		if w >= lipgloss.Width(welcome) {
			fmt.Println(logoRenderer())
			fmt.Print(welcome, "\n\n")
		} else {
			fmt.Print("Welcome to the boot.dev CLI!\n\n")
		}

		fmt.Println("Please navigate to:\n" +
			viper.GetString("base_url") +
			"/cli/login")

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
