//go:build ignore

package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/blang/semver"
)

func getTag(match ...string) (string, *semver.PRVersion) {
	args := append([]string{
		"describe", "--tags",
	}, match...)
	if tag, err := exec.Command("git", args...).Output(); err != nil {
		return "", nil
	} else {
		tagParts := strings.Split(string(tag), "-")
		if len(tagParts) == 3 {
			ahead, err := semver.NewPRVersion(tagParts[1])
			if err == nil {
				return tagParts[0], &ahead
			}
			log.Printf("semver.NewPRVersion(%s): %v", tagParts[1], err)
		} else if len(tagParts) == 4 {
			ahead, err := semver.NewPRVersion(tagParts[2])
			if err == nil {
				return tagParts[0] + "-" + tagParts[1], &ahead
			}
			log.Printf("semver.NewPRVersion(%s): %v", tagParts[2], err)
		}

		return string(tag), nil
	}
}

func main() {
	// Find the last vX.X.X Tag and get how many builds we are ahead of it.
	versionStr, ahead := getTag("--match", "v*")
	version, err := semver.ParseTolerant(versionStr)
	if err != nil {
		// no version tag found so just return what ever we can find.
		defaultVersionStr := "0.0.0-unknown"
		log.Printf("semver.ParseTolerant(%s): %v, defaulting to %s", versionStr, err, defaultVersionStr)
		fmt.Println(defaultVersionStr)
		return
	}
	if ahead == nil {
		// Seems that we are going to build a release.
		// So the version number should already be correct.
		fmt.Println(version.String())
		return
	}

	// Get the tag of the current revision.
	tag, _ := getTag("--exact-match")

	// If we don't have any tag assume "dev"
	if tag == "" || strings.HasPrefix(tag, "nightly") {
		tag = "dev"
	}
	// Get the most likely next version:
	if !strings.Contains(version.String(), "rc") {
		version.Patch = version.Patch + 1
	}

	if pr, err := semver.NewPRVersion(tag); err == nil {
		// append the tag as pre-release name
		version.Pre = append(version.Pre, pr)
	} else {
		log.Printf("semver.NewPRVersion(%s): %v", tag, err)
	}

	// append how many commits we are ahead of the last release
	version.Pre = append(version.Pre, *ahead)

	fmt.Println(version.String())
}
