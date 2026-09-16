package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
)

var commandNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// NormalizeCommandName converts a package command name to its canonical form
// and rejects values that cannot safely be used as a command identifier.
func NormalizeCommandName(name string) (string, error) {
	normalized := strings.ToLower(name)
	if !commandNamePattern.MatchString(normalized) {
		return "", fmt.Errorf("%q must match [A-Za-z0-9][A-Za-z0-9_-]*", name)
	}

	return normalized, nil
}

// Self ...
func Self() string {
	return filepath.Base(os.Args[0])
}

// GetAkamaiCliPath returns the "$AKAMAI_CLI_HOME/.akamai-cli" value and tries to create it if not existing.
//
// Errors out if:
//
// * $AKAMAI_CLI_HOME is not defined
//
// * $AKAMAI_CLI_HOME/.akamai-cli does not exist, and we cannot create it
func GetAkamaiCliPath() (string, error) {
	cliHome := os.Getenv("AKAMAI_CLI_HOME")
	if cliHome == "" {
		var err error
		cliHome, err = homedir.Dir()
		if err != nil {
			return "", cli.Exit("Package install directory could not be found. Please set $AKAMAI_CLI_HOME.", -1)
		}
	}

	cliPath := filepath.Join(cliHome, ".akamai-cli")
	err := os.MkdirAll(cliPath, 0700)
	if err != nil {
		return "", cli.Exit("Unable to create Akamai CLI root directory.", -1)
	}

	return cliPath, nil
}

// GetAkamaiCliSrcPath returns $AKAMAI_CLI_HOME/.akamai-cli/src
func GetAkamaiCliSrcPath() (string, error) {
	cliHome, err := GetAkamaiCliPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(cliHome, "src"), nil
}

// GetAkamaiCliVenvPath - returns the .akamai-cli/venv path, for Python virtualenv
func GetAkamaiCliVenvPath() (string, error) {
	cliHome, err := GetAkamaiCliPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(cliHome, "venv"), nil
}

// GetPkgVenvPath - returns the package virtualenv path
func GetPkgVenvPath(pkgName string) (string, error) {
	vePath, err := GetAkamaiCliVenvPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(vePath, pkgName), nil
}

// Githubize returns the GitHub package repository URI
func Githubize(repo string) string {
	if strings.HasPrefix(repo, "http") || strings.HasPrefix(repo, "ssh") || strings.HasSuffix(repo, ".git") {
		return strings.TrimPrefix(repo, "ssh://")
	}

	if strings.HasPrefix(repo, "file://") {
		return repo
	}

	if !strings.Contains(repo, "/") {
		repo = "akamai/cli-" + strings.TrimPrefix(repo, "cli-")
	}

	// Handle Github migration from akamai-open -> akamai
	if strings.HasPrefix(repo, "akamai-open/") {
		repo = "akamai/" + strings.TrimPrefix(repo, "akamai-open/")
	}

	return "https://github.com/" + repo + ".git"
}

// CapitalizeFirstWord capitalizes only first character in the string
func CapitalizeFirstWord(str string) string {
	if len(str) <= 1 {
		return strings.ToUpper(str)
	}
	return strings.ToUpper(string(str[0])) + str[1:]
}

// InsertAfterNthWord inserts one string into another after the nth word specified by index
func InsertAfterNthWord(s, val string, index int) string {
	words := strings.Fields(s)
	if len(words) <= index {
		return strings.Join(append(words, val), " ")
	}
	words = append(words[:index+1], words[index:]...)
	words[index] = val
	return strings.Join(words, " ")
}
