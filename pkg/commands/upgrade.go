//+build !noautoupgrade

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

package commands

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/fatih/color"
	"github.com/inconshreveable/go-update"
	"github.com/kardianos/osext"

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/version"
)

// CheckUpgradeVersion ...
func CheckUpgradeVersion(ctx context.Context, force bool) string {
	term := terminal.Get(ctx)

	if !term.IsTTY() {
		return ""
	}

	data := strings.TrimSpace(config.GetConfigValue("cli", "last-upgrade-check"))

	if data == "ignore" {
		return ""
	}

	checkForUpgrade := false
	if data == "never" || force {
		checkForUpgrade = true
	}

	if !checkForUpgrade {
		configValue := strings.TrimPrefix(strings.TrimSuffix(data, "\""), "\"")
		lastUpgrade, err := time.Parse(time.RFC3339, configValue)

		if err != nil {
			return ""
		}

		currentTime := time.Now()
		if lastUpgrade.Add(sleepTime24Hours).Before(currentTime) {
			checkForUpgrade = true
		}
	}

	if checkForUpgrade {
		config.SetConfigValue("cli", "last-upgrade-check", time.Now().Format(time.RFC3339))
		err := config.SaveConfig(ctx)
		if err != nil {
			return ""
		}

		latestVersion := getLatestReleaseVersion()
		if version.Compare(version.Version, latestVersion) == 1 {
			if !force {
				answer, _ := term.Confirm(fmt.Sprintf(
					"New upgrade found: %s (you are running: %s). Upgrade now? [Y/n]: ",
					color.BlueString(latestVersion),
					color.BlueString(version.Version),
				), true)

				if !answer {
					return ""
				}
			}
			return latestVersion
		}
	}

	return ""
}

func getLatestReleaseVersion() string {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Head("https://github.com/akamai/cli/releases/latest")
	if err != nil {
		return "0"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return "0"
	}

	location := resp.Header.Get("Location")
	latestVersion := filepath.Base(location)

	return latestVersion
}

// UpgradeCli ...
func UpgradeCli(ctx context.Context, latestVersion string) bool {
	term := terminal.Get(ctx)

	term.Spinner().Start("Upgrading Akamai CLI")

	cmd := command{
		Version: latestVersion,
		Bin:     "https://github.com/akamai/cli/releases/download/{{.Version}}/akamai-{{.Version}}-{{.OS}}{{.Arch}}{{.BinSuffix}}",
		Arch:    runtime.GOARCH,
		OS:      runtime.GOOS,
	}

	if runtime.GOOS == "darwin" {
		cmd.OS = "mac"
	}

	if runtime.GOOS == "windows" {
		cmd.BinSuffix = ".exe"
	}

	t := template.Must(template.New("url").Parse(cmd.Bin))
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, cmd); err != nil {
		return false
	}

	resp, err := http.Get(buf.String())
	if err != nil || resp.StatusCode != http.StatusOK {
		term.Spinner().Fail()
		term.Writeln(color.RedString("Unable to download release, please try again."))
		return false
	}
	defer resp.Body.Close()

	shaResp, err := http.Get(fmt.Sprintf("%v%v", buf.String(), ".sig"))
	if err != nil || shaResp.StatusCode != http.StatusOK {
		term.Spinner().Fail()
		term.Writeln(color.RedString("Unable to retrieve signature for verification, please try again."))
		return false
	}
	defer shaResp.Body.Close()

	shabody, err := ioutil.ReadAll(shaResp.Body)
	if err != nil {
		term.Spinner().Fail()
		term.Writeln(color.RedString("Unable to retrieve signature for verification, please try again."))
		return false
	}

	shasum, err := hex.DecodeString(strings.TrimSpace(string(shabody)))
	if err != nil {
		term.Spinner().Fail()
		term.Writeln(color.RedString("Unable to retrieve signature for verification, please try again."))
		return false
	}

	selfPath, err := osext.Executable()
	if err != nil {
		term.Spinner().Fail()
		term.Writeln(color.RedString("Unable to determine install location"))
		return false
	}

	err = update.Apply(resp.Body, update.Options{TargetPath: selfPath, Checksum: shasum})
	if err != nil {
		term.Spinner().Fail()
		if rerr := update.RollbackError(err); rerr != nil {
			term.Writeln(color.RedString("Unable to install or rollback, please re-install."))
			os.Exit(1)
			return false
		} else if strings.HasPrefix(err.Error(), "Upgrade file has wrong checksum.") {
			term.Writeln(color.RedString(err.Error()))
			term.Writeln(color.RedString("Checksums do not match, please try again."))
		}
		return false
	}

	term.Spinner().OK()

	if err == nil {
		os.Args[0] = selfPath
	}
	err = passthruCommand(os.Args)
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)

	return true
}

func getUpgradeCommand() *subcommands {
	return &subcommands{
		Commands: []command{
			{
				Name:        "upgrade",
				Description: "Upgrade Akamai CLI to the latest version",
			},
		},
		Action: cmdUpgrade,
	}
}
