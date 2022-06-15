//go:build !noautoupgrade
// +build !noautoupgrade

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

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/version"
	"github.com/fatih/color"
	"github.com/inconshreveable/go-update"
	"github.com/urfave/cli/v2"
)

// CheckUpgradeVersion ...
func CheckUpgradeVersion(ctx context.Context, force bool) string {
	term := terminal.Get(ctx)
	cfg := config.Get(ctx)

	if !term.IsTTY() {
		return ""
	}

	data, _ := cfg.GetValue("cli", "last-upgrade-check")
	data = strings.TrimSpace(data)

	if data == "ignore" && !force {
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
		if lastUpgrade.Add(sleep24HDuration).Before(currentTime) {
			checkForUpgrade = true
		}
	}

	if checkForUpgrade {
		cfg.SetValue("cli", "last-upgrade-check", time.Now().Format(time.RFC3339))
		err := cfg.Save(ctx)
		if err != nil {
			return ""
		}

		latestVersion := getLatestReleaseVersion(ctx)
		comp := version.Compare(version.Version, latestVersion)
		if comp == version.Smaller {
			term.Spinner().Stop(terminal.SpinnerStatusOK)
			_, _ = term.Writeln("You can find more details about the new version here: https://github.com/akamai/cli/releases")
			if answer, err := term.Confirm(fmt.Sprintf(
				"New update found: %s. You are running: %s. Upgrade now?",
				color.BlueString(latestVersion),
				color.BlueString(version.Version),
			), true); err != nil || !answer {
				return ""
			}
			return latestVersion
		}
		if comp == version.Equals {
			return version.Version
		}
	}

	return ""
}

func getLatestReleaseVersion(ctx context.Context) string {
	logger := log.FromContext(ctx)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	repo := "https://github.com/akamai/cli"
	if r := os.Getenv("CLI_REPOSITORY"); r != "" {
		repo = r
	}
	resp, err := client.Head(fmt.Sprintf("%s/releases/latest", repo))
	if err != nil {
		return "0"
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error(err.Error())
		}
	}()

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
	logger := log.FromContext(ctx)

	term.Spinner().Start("Upgrading Akamai CLI")

	repo := "https://github.com/akamai/cli"
	if r := os.Getenv("CLI_REPOSITORY"); r != "" {
		repo = r
	}
	cmd := command{
		Version: latestVersion,
		Bin:     fmt.Sprintf("%s/releases/download/{{.Version}}/akamai-{{.Version}}-{{.OS}}{{.Arch}}{{.BinSuffix}}", repo),
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
		errMsg := color.RedString("Unable to download release, please try again.")
		term.Writeln(errMsg)
		logger.Error(errMsg)
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error(err.Error())
		}
	}()

	shaResp, err := http.Get(fmt.Sprintf("%v%v", buf.String(), ".sig"))
	if err != nil || shaResp.StatusCode != http.StatusOK {
		term.Spinner().Fail()
		term.Writeln(color.RedString("Unable to retrieve signature for verification, please try again."))
		return false
	}
	defer func() {
		if err := shaResp.Body.Close(); err != nil {
			logger.Error(err.Error())
		}
	}()

	shaBody, err := ioutil.ReadAll(shaResp.Body)
	if err != nil {
		term.Spinner().Fail()
		term.Writeln(color.RedString("Unable to retrieve signature for verification, please try again."))
		return false
	}

	shaSum, err := hex.DecodeString(strings.TrimSpace(string(shaBody)))
	if err != nil {
		term.Spinner().Fail()
		term.Writeln(color.RedString("Unable to retrieve signature for verification, please try again."))
		return false
	}

	selfPath := os.Args[0]

	err = update.Apply(resp.Body, update.Options{TargetPath: selfPath, Checksum: shaSum})
	if err != nil {
		term.Spinner().Fail()
		if rerr := update.RollbackError(err); rerr != nil {
			term.Writeln(color.RedString("Unable to install or rollback, please re-install."))
			os.Exit(1)
			return false
		} else if strings.HasPrefix(err.Error(), "Upgrade file has wrong checksum.") {
			term.Writeln(color.RedString(err.Error()))
			term.Writeln(color.RedString("Checksums do not match, please try again."))
			return false
		}
		term.Writeln(color.RedString(err.Error()))
		return false
	}

	term.Spinner().OK()

	if err == nil {
		os.Args[0] = selfPath
	}

	subCmd := createCommand(os.Args[0], os.Args[1:])
	if err = passthruCommand(ctx, subCmd, packages.NewLangManager(), packages.LanguageRequirements{}, selfPath); err != nil {
		cli.OsExiter(1)
		return false
	}
	cli.OsExiter(0)

	return true
}
