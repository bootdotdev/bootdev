package checks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"

	"golang.org/x/mod/semver"
)

const repoOwner = "bootdotdev"
const repoName = "bootdev"

type GHTag struct {
	Name string `json:"name"`
}

func PromptUpdateIfNecessary(currentVersion string) error {
	latest, err := getLatestVersion()
	if err != nil {
		return err
	}
	isUpdateRequired := isUpdateRequired(currentVersion, latest)
	isOutdated := isOutdated(currentVersion, latest)

	if isOutdated {
		fmt.Fprintln(os.Stderr, "A new version of the bootdev CLI is available!")
		fmt.Fprintln(os.Stderr, "Please run the following command to update:")
		fmt.Fprintf(os.Stderr, "  go install github.com/%s/%s@latest\n\n", repoOwner, repoName)
		if isUpdateRequired {
			os.Exit(1)
		}
	}
	return nil
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
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", repoOwner, repoName))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tags []GHTag
	err = json.Unmarshal(body, &tags)
	if err != nil {
		return "", err
	}

	sort.Slice(tags, func(i, j int) bool {
		return semver.Compare(tags[j].Name, tags[i].Name) == -1
	})

	if len(tags) == 0 {
		return "", errors.New("no tags found")
	}

	return tags[0].Name, nil
}
