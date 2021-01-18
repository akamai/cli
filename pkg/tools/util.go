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
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"strings"
)

func Self() string {
	return filepath.Base(os.Args[0])
}

func GetAkamaiCliPath() (string, error) {
	cliHome := os.Getenv("AKAMAI_CLI_HOME")
	if cliHome == "" {
		var err error
		cliHome, err = homedir.Dir()
		if err != nil {
			return "", cli.NewExitError("Package install directory could not be found. Please set $AKAMAI_CLI_HOME.", -1)
		}
	}

	cliPath := filepath.Join(cliHome, ".akamai-cli")
	err := os.MkdirAll(cliPath, 0700)
	if err != nil {
		return "", cli.NewExitError("Unable to create Akamai CLI root directory.", -1)
	}

	return cliPath, nil
}

func GetAkamaiCliSrcPath() (string, error) {
	cliHome, _ := GetAkamaiCliPath()

	return filepath.Join(cliHome, "src"), nil
}

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
