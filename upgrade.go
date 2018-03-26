//+build !noautoupgrade

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
	"bytes"
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

	akamai "github.com/akamai/cli-common-golang"
	"github.com/fatih/color"
	"github.com/inconshreveable/go-update"
	"github.com/kardianos/osext"
	"github.com/mattn/go-isatty"
)

func checkForUpgrade(force bool) string {
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return ""
	}

	data := strings.TrimSpace(getConfigValue("cli", "last-upgrade-check"))

	if data == "ignore" {
		return ""
	}

	checkForUpgrade := false
	if data == "never" || force {
		checkForUpgrade = true
	}

	if !checkForUpgrade {
		configValue := strings.TrimPrefix(strings.TrimSuffix(string(data), "\""), "\"")
		lastUpgrade, err := time.Parse(time.RFC3339, configValue)

		if err != nil {
			return ""
		}

		currentTime := time.Now()
		if lastUpgrade.Add(time.Hour * 24).Before(currentTime) {
			checkForUpgrade = true
		}
	}

	if checkForUpgrade {
		setConfigValue("cli", "last-upgrade-check", time.Now().Format(time.RFC3339))
		err := saveConfig()
		if err != nil {
			return ""
		}

		latestVersion := getLatestReleaseVersion()
		if versionCompare(VERSION, latestVersion) == 1 {
			if !force {
				fmt.Fprintf(
					akamai.App.Writer,
					"New upgrade found: %s (you are running: %s). Upgrade now? [Y/n]: ",
					color.BlueString(latestVersion),
					color.BlueString(VERSION),
				)
				answer := ""
				fmt.Scanln(&answer)
				if answer != "" && strings.ToLower(answer) != "y" {
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

	if resp.StatusCode != 302 {
		return "0"
	}

	location := resp.Header.Get("Location")
	latestVersion := filepath.Base(location)

	return latestVersion
}

func upgradeCli(latestVersion string) bool {
	akamai.StartSpinner("Upgrading Akamai CLI", "Upgrading Akamai CLI...... ["+color.GreenString("OK")+"]\n\n")

	cmd := Command{
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
		trackEvent("upgrade.failed", "to: "+latestVersion+" from:"+VERSION)
		return false
	}

	url := buf.String()

	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil || resp.StatusCode != 200 {
		akamai.StopSpinnerFail()
		fmt.Fprintln(akamai.App.ErrWriter, color.RedString("Unable to download release, please try again."))
		trackEvent("upgrade.failed", "to: "+latestVersion+" from:"+VERSION)
		return false
	}

	shaResp, err := http.Get(url + ".sig")
	defer shaResp.Body.Close()
	if err != nil || shaResp.StatusCode != 200 {
		akamai.StopSpinnerFail()
		fmt.Fprintln(akamai.App.ErrWriter, color.RedString("Unable to retrieve signature for verification, please try again."))
		trackEvent("upgrade.failed", "to: "+latestVersion+" from:"+VERSION)
		return false
	}

	shabody, err := ioutil.ReadAll(shaResp.Body)
	if err != nil {
		akamai.StopSpinnerFail()
		fmt.Fprintln(akamai.App.ErrWriter, color.RedString("Unable to retrieve signature for verification, please try again."))
		trackEvent("upgrade.failed", "to: "+latestVersion+" from:"+VERSION)
		return false
	}

	shasum, err := hex.DecodeString(strings.TrimSpace(string(shabody)))
	if err != nil {
		akamai.StopSpinnerFail()
		fmt.Fprintln(akamai.App.ErrWriter, color.RedString("Unable to retrieve signature for verification, please try again."))
		trackEvent("upgrade.failed", "to: "+latestVersion+" from:"+VERSION)
		return false
	}

	selfPath, err := osext.Executable()
	if err != nil {
		akamai.StopSpinnerFail()
		fmt.Fprintln(akamai.App.ErrWriter, color.RedString("Unable to determine install location"))
		trackEvent("upgrade.failed", "to: "+latestVersion+" from:"+VERSION)
		return false
	}

	err = update.Apply(resp.Body, update.Options{TargetPath: selfPath, Checksum: shasum})
	if err != nil {
		akamai.StopSpinnerFail()
		if rerr := update.RollbackError(err); rerr != nil {
			fmt.Fprintln(akamai.App.ErrWriter, color.RedString("Unable to install or rollback, please re-install."))
			trackEvent("upgrade.failed", "to: "+latestVersion+" from:"+VERSION)
			os.Exit(1)
			return false
		} else if strings.HasPrefix(err.Error(), "Upgrade file has wrong checksum.") {
			fmt.Fprintln(akamai.App.ErrWriter, color.RedString(err.Error()))
			fmt.Fprintln(akamai.App.ErrWriter, color.RedString("Checksums do not match, please try again."))
		}
		trackEvent("upgrade.failed", "to: "+latestVersion+" from:"+VERSION)
		return false
	}

	trackEvent("upgrade.success", "to: "+latestVersion+" from:"+VERSION)
	akamai.StopSpinnerOk()

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

func getUpgradeCommand() *commandPackage {
	return &commandPackage{
		Commands: []Command{
			{
				Name:        "upgrade",
				Description: "Upgrade Akamai CLI to the latest version",
			},
		},
		action: cmdUpgrade,
	}
}
