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
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/version"
	"github.com/fatih/color"
)

// CheckUpgradeVersion ...
func CheckUpgradeVersion(ctx context.Context, force bool) string {
	term := terminal.Get(ctx)
	cfg := config.Get(ctx)

	if !term.IsTTY() {
		return ""
	}

	term.Spinner().Start("Checking for upgrades...")

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
