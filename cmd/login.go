package cmd

import (
	"bufio"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

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
		welcome := lipgloss.PlaceHorizontal(lipgloss.Width(logo), lipgloss.Center, "Welcome to the Boot.dev CLI!")

		if w >= lipgloss.Width(welcome) {
			fmt.Print(logo)
			fmt.Print(welcome, "\n\n")
		} else {
			fmt.Print("Welcome to the Boot.dev CLI!\n\n")
		}

		loginURL := viper.GetString("frontend_url") + "/cli/login"

		fmt.Println("Please navigate to:\n" + loginURL)

		inputChan := make(chan string, 1)

		go func() {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("\nPaste your login code: ")
			text, _ := reader.ReadString('\n')
			inputChan <- text
		}()

		stopHTTPServer := startHTTPServer(inputChan)
		defer stopHTTPServer()

		// attempt to open the browser
		go func() {
			browser.Stdout = nil
			browser.Stderr = nil
			browser.OpenURL(loginURL)
		}()

		// race the web server against the user's input
		text := <-inputChan
		stopHTTPServer()

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

const maxLoginCodeBytes = 4 * 1024

func cors(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allowedOrigin == "" || r.Header.Get("Origin") != allowedOrigin {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func frontendOrigin(frontendURL string) string {
	parsed, err := url.Parse(frontendURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func newLoginHTTPHandler(inputChan chan<- string, frontendURL string) http.Handler {
	mux := http.NewServeMux()
	allowedOrigin := frontendOrigin(frontendURL)

	handleSubmit := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST, OPTIONS")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxLoginCodeBytes)
		code, err := io.ReadAll(r.Body)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				http.Error(w, "login code too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "failed to read login code", http.StatusBadRequest)
			return
		}
		select {
		case inputChan <- string(code):
		default:
			http.Error(w, "login code already received", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		// Clear current line
		fmt.Print("\n\033[1A\033[K")
	}

	handleHealth := func(w http.ResponseWriter, r *http.Request) {
		// 200 OK
	}

	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		loginURL := frontendURL + "/cli/login"
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
	}

	mux.Handle("/submit", cors(allowedOrigin, http.HandlerFunc(handleSubmit)))
	mux.Handle("/health", cors(allowedOrigin, http.HandlerFunc(handleHealth)))
	mux.Handle("/{$}", http.HandlerFunc(handleRedirect))
	return mux
}

func startHTTPServer(inputChan chan<- string) func() {
	server := &http.Server{
		Handler:           newLoginHTTPHandler(inputChan, viper.GetString("frontend_url")),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       5 * time.Second,
	}

	listener, err := net.Listen("tcp", "localhost:9417")
	if err != nil {
		return func() {}
	}

	go func() {
		_ = server.Serve(listener)
	}()

	var stopOnce sync.Once
	return func() {
		stopOnce.Do(func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = server.Shutdown(ctx)
		})
	}
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
