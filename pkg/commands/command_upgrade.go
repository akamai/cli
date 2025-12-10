//go:build !noautoupgrade

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
	"fmt"
	"os"
	"time"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/version"
	"github.com/urfave/cli/v2"
)

func cmdUpgrade(c *cli.Context) error {
	c.Context = log.WithCommandContext(c.Context, c.Command.Name)
	logger := log.FromContext(c.Context)
	start := time.Now()
	logger.Debug("UPGRADE START")
	defer func() {
		logger.Debug(fmt.Sprintf("UPGRADE FINISH: %v", time.Since(start)))
	}()
	term := terminal.Get(c.Context)

	term.Spinner().Start("Checking for upgrades...")

	latestVersion := CheckUpgradeVersion(c.Context, true)
	if latestVersion != "" && latestVersion != version.Version {
		term.Spinner().Stop(terminal.SpinnerStatusOK)
		os.Args = []string{os.Args[0], "--version"}
		return UpgradeCli(c.Context, latestVersion)
	}
	term.Spinner().Stop(terminal.SpinnerStatusWarnOK)
	if latestVersion == version.Version {
		term.Printf("Akamai CLI (%s) is already up-to-date", color.CyanString("v"+version.Version))
		return nil
	}
	if latestVersion != "" {
		term.Printf("Akamai CLI version: %s", color.CyanString("v"+version.Version))
	}
	return nil
}
