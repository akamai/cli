//go:build noautoupgrade
// +build noautoupgrade

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

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func CheckUpgradeVersion(ctx context.Context, force bool) string {
	return ""
}

func getLatestReleaseVersion() string {
	return "0"
}

func UpgradeCli(ctx context.Context, latestVersion string) bool {
	return false
}

func getUpgradeCommand() *cli.Command {
	return &cli.Command{
		Name:        "upgrade",
		Description: "Upgrade Akamai CLI to the latest version",
		Action: func(_ *cli.Context) error {
			return cli.Exit(color.RedString("Upgrade command is not available for your installation. If you installed Akamai CLI with Homebrew, please run 'brew upgrade akamai' in order to perform upgrade."), 1)
		},
	}
}
