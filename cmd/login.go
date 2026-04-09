package cmd

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func logoRenderer() string {
	return logo
}

//go:embed boots.txt
var logo string

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
		welcome := lipgloss.PlaceHorizontal(lipgloss.Width(logoRenderer()), lipgloss.Center, "Welcome to the Boot.dev CLI!")

		if w >= lipgloss.Width(welcome) {
			fmt.Print(logoRenderer())
			fmt.Print(welcome, "\n\n")
		} else {
			fmt.Print("Welcome to the Boot.dev CLI!\n\n")
		}

		loginUrl := viper.GetString("frontend_url") + "/cli/login"

		fmt.Println("Please navigate to:\n" + loginUrl)

		inputChan := make(chan string)

		go func() {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("\nPaste your login code: ")
			text, _ := reader.ReadString('\n')
			inputChan <- text
		}()

		go func() {
			startHTTPServer(inputChan)
		}()

		// attempt to open the browser
		go func() {
			browser.Stdout = nil
			browser.Stderr = nil
			browser.OpenURL(loginUrl)
		}()

		// race the web server against the user's input
		text := <-inputChan

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

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

func startHTTPServer(inputChan chan string) {
	handleSubmit := func(w http.ResponseWriter, r *http.Request) {
		code, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		inputChan <- string(code)
		// Clear current line
		fmt.Print("\n\033[1A\033[K")
	}

	handleHealth := func(w http.ResponseWriter, r *http.Request) {
		// 200 OK
	}

	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		loginUrl := viper.GetString("frontend_url") + "/cli/login"
		http.Redirect(w, r, loginUrl, http.StatusSeeOther)
	}

	http.Handle("POST /submit", cors(http.HandlerFunc(handleSubmit)))
	http.Handle("/health", cors(http.HandlerFunc(handleHealth)))
	http.Handle("/{$}", cors(http.HandlerFunc(handleRedirect)))

	// if we fail, oh well. we fall back to entering the code
	_ = http.ListenAndServe("localhost:9417", nil)
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
