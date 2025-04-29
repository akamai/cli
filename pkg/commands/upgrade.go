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

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/config"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/version"
)

type versionProvider interface {
	getLatestReleaseVersion(ctx context.Context) string
	getCurrentVersion() string
}

// CheckUpgradeVersion ...
func CheckUpgradeVersion(ctx context.Context, force bool) string {
	return checkUpgradeVersion(ctx, force, defaultVersionProvider{})
}

func checkUpgradeVersion(ctx context.Context, force bool, provider versionProvider) string {
	term := terminal.Get(ctx)
	cfg := config.Get(ctx)
	logger := log.FromContext(ctx)

	if !term.IsTTY() {
		return ""
	}

	logger.Debug("Checking for upgrades")

	data, _ := cfg.GetValue("cli", "last-upgrade-check")
	data = strings.TrimSpace(data)
	if data == "ignore" && !force {
		logger.Error("Upgrade checks are disabled")
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
			logger.Error(fmt.Sprintf("Error parsing last upgrade check time: %v", err))
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
			logger.Error(fmt.Sprintf("Error saving config: %v", err))
			return ""
		}

		latestVersion := provider.getLatestReleaseVersion(ctx)
		currentVersion := provider.getCurrentVersion()
		comp := version.Compare(currentVersion, latestVersion)
		if comp == version.Smaller {
			term.Spinner().Stop(terminal.SpinnerStatusOK)
			_, _ = term.Writeln("You can find more details about the new version here: https://github.com/akamai/cli/releases")
			if answer, err := term.Confirm(fmt.Sprintf(
				"New update found: %s. You are running: %s. Upgrade now?",
				color.BlueString(latestVersion),
				color.BlueString(currentVersion),
			), true); err != nil || !answer {
				logger.Error(fmt.Sprintf("Upgrade declined: %v", err))
				return ""
			}
			return latestVersion
		}
		if comp == version.Equals {
			// A non-empty version is returned but the caller checks whether latest == current
			// and does not perform an upgrade in such case.
			return currentVersion
		}
	}

	return ""
}

type defaultVersionProvider struct{}

func (p defaultVersionProvider) getLatestReleaseVersion(ctx context.Context) string {
	logger := log.FromContext(ctx)
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	repo := "https://github.com/akamai/cli"
	if r := os.Getenv("CLI_REPOSITORY"); r != "" {
		repo = r
	}
	resp, err := client.Head(fmt.Sprintf("%s/releases/latest", repo))
	if err != nil {
		logger.Error(fmt.Sprintf("Error checking for latest version: %v", err))
		return "0"
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error(fmt.Sprintf("Error closing response body: %v", err))
		}
	}()

	if resp.StatusCode != http.StatusFound {
		logger.Error(fmt.Sprintf("Error checking for latest version: %s", resp.Status))
		return "0"
	}

	location := resp.Header.Get("Location")
	latestVersion := filepath.Base(location)

	return latestVersion
}

func (p defaultVersionProvider) getCurrentVersion() string {
	return version.Version
}
