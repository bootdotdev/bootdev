package version

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"golang.org/x/mod/semver"
)

const (
	repoOwner = "bootdotdev"
	repoName  = "bootdev"
)

type VersionInfo struct {
	CurrentVersion   string
	LatestVersion    string
	IsOutdated       bool
	IsUpdateRequired bool
	FailedToFetch    error
}

func FetchUpdateInfo(currentVersion string) VersionInfo {
	latest, err := getLatestVersion()
	if err != nil {
		return VersionInfo{
			FailedToFetch: err,
		}
	}
	isUpdateRequired := isUpdateRequired(currentVersion, latest)
	isOutdated := isOutdated(currentVersion, latest)
	return VersionInfo{
		IsUpdateRequired: isUpdateRequired,
		IsOutdated:       isOutdated,
		CurrentVersion:   currentVersion,
		LatestVersion:    latest,
	}
}

func (v *VersionInfo) PromptUpdateIfAvailable() {
	if v.IsOutdated {
		fmt.Fprintln(os.Stderr, "A new version of the bootdev CLI is available!")
		fmt.Fprintln(os.Stderr, "Please run the following command to update:")
		fmt.Fprintln(os.Stderr, "  bootdev upgrade")
		fmt.Fprintln(os.Stderr, "or")
		fmt.Fprintf(os.Stderr, "  go install github.com/bootdotdev/bootdev@%s\n\n", v.LatestVersion)
	}
}

// Returns true if the current version is older than the latest.
func isOutdated(current string, latest string) bool {
	return semver.Compare(current, latest) < 0
}

// Returns true if the latest version has a higher major or minor
// number than the current version. If you don't want to force
// an update, you can increment the patch number instead.
func isUpdateRequired(current string, latest string) bool {
	latestMajorMinor := semver.MajorMinor(latest)
	currentMajorMinor := semver.MajorMinor(current)
	return semver.Compare(currentMajorMinor, latestMajorMinor) < 0
}

func getLatestVersion() (string, error) {
	goproxyDefault := "https://proxy.golang.org"
	goproxy := goproxyDefault
	cmd := exec.Command("go", "env", "GOPROXY")
	output, err := cmd.Output()
	if err == nil {
		goproxy = strings.TrimSpace(string(output))
	}

	proxies := strings.Split(goproxy, ",")
	if !slices.Contains(proxies, goproxyDefault) {
		proxies = append(proxies, goproxyDefault)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	var lastErr error
	for _, proxy := range proxies {
		proxy = strings.TrimSpace(proxy)
		proxy = strings.TrimRight(proxy, "/")
		if proxy == "direct" || proxy == "off" {
			continue
		}

		url := fmt.Sprintf("%s/github.com/%s/%s/@latest", proxy, repoOwner, repoName)
		version, err := fetchLatestWithRetry(client, url)
		if err == nil {
			return version, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return "", fmt.Errorf("failed to fetch latest version: %w", lastErr)
	}
	return "", fmt.Errorf("failed to fetch latest version")
}

// fetchLatestWithRetry queries a single proxy, retrying a few times because
// proxy.golang.org occasionally resets the connection or returns a transient
// 5xx. A single drop should not fail the whole version check.
func fetchLatestWithRetry(client *http.Client, url string) (string, error) {
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		version, err := fetchLatestFromProxy(client, url)
		if err == nil {
			return version, nil
		}
		lastErr = err
		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
		}
	}
	return "", lastErr
}

func fetchLatestFromProxy(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s from %s", resp.Status, url)
	}

	var version struct{ Version string }
	if err := json.Unmarshal(body, &version); err != nil {
		return "", fmt.Errorf("invalid response from %s: %w", url, err)
	}
	if version.Version == "" {
		return "", fmt.Errorf("empty version in response from %s", url)
	}

	return version.Version, nil
}
