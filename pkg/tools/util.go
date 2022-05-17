// Copyright 2018. Akamai Technologies, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
)

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
