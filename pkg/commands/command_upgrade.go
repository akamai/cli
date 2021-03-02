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
	"github.com/akamai/cli/pkg/log"
	"os"
	"time"

	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/version"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func cmdUpgrade(c *cli.Context) error {
	c.Context = log.WithCommandContext(c.Context, c.Command.Name)
	logger := log.WithCommand(c.Context, c.Command.Name)
	start := time.Now()
	logger.Debug("UPGRADE START")
	defer func() {
		logger.Debugf("UPGRADE FINISH: %v", time.Now().Sub(start))
	}()
	term := terminal.Get(c.Context)

	term.Spinner().Start("Checking for upgrades...")

	latestVersion := CheckUpgradeVersion(c.Context, true)
	if latestVersion != "" && latestVersion != version.Version {
		os.Args = []string{os.Args[0], "--version"}
		success := UpgradeCli(c.Context, latestVersion)
		if success {
			stats.TrackEvent(c.Context, "upgrade.user", "success", "to: "+latestVersion+" from:"+version.Version)
		} else {
			stats.TrackEvent(c.Context, "upgrade.user", "failed", "to: "+latestVersion+" from:"+version.Version)
		}
		return nil
	}
	term.Spinner().Stop(terminal.SpinnerStatusWarnOK)
	if latestVersion == version.Version {
		term.Printf("Akamai CLI (%s) is already up-to-date", color.CyanString("v"+version.Version))
		return nil
	}
	term.Printf("Akamai CLI version: %s", color.CyanString("v"+version.Version))
	return nil
}
