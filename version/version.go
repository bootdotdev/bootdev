package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"

	"golang.org/x/mod/semver"
)

const repoOwner = "bootdotdev"
const repoName = "bootdev"

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

	for _, proxy := range proxies {
		proxy = strings.TrimSpace(proxy)
		proxy = strings.TrimRight(proxy, "/")
		if proxy == "direct" || proxy == "off" {
			continue
		}

		url := fmt.Sprintf("%s/github.com/%s/%s/@latest", proxy, repoOwner, repoName)
		resp, err := http.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		var version struct{ Version string }
		if err = json.Unmarshal(body, &version); err != nil {
			continue
		}

		return version.Version, nil
	}

	return "", fmt.Errorf("failed to fetch latest version")
}
