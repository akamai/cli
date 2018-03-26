/*
 Copyright 2018. Akamai Technologies, Inc

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package main

import (
	"fmt"
	"os"

	akamai "github.com/akamai/cli-common-golang"
	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func cmdUpgrade(c *cli.Context) error {
	akamai.StartSpinner("Checking for upgrades...", "Checking for upgrades...... ["+color.GreenString("OK")+"]\n")

	if latestVersion := checkForUpgrade(true); latestVersion != "" {
		akamai.StopSpinnerOk()
		fmt.Fprintf(akamai.App.Writer, "Found new version: %s (current version: %s)\n", color.BlueString("v"+latestVersion), color.BlueString("v"+VERSION))
		os.Args = []string{os.Args[0], "--version"}
		success := upgradeCli(latestVersion)
		if success {
			trackEvent("upgrade.success", "to: "+latestVersion+" from:"+VERSION)
		} else {
			trackEvent("upgrade.failed", "to: "+latestVersion+" from:"+VERSION)
		}
	} else {
		akamai.StopSpinnerWarnOk()
		fmt.Fprintf(akamai.App.Writer, "Akamai CLI (%s) is already up-to-date", color.CyanString("v"+VERSION))
	}

	return nil
}
