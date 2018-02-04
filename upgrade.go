//+build !noautoupgrade

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/fatih/color"
	"github.com/inconshreveable/go-update"
	"github.com/kardianos/osext"
	"github.com/mattn/go-isatty"
)

func checkForUpgrade(force bool) string {
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return ""
	}

	data := strings.TrimSpace(getConfigValue("last-upgrade-check"))

	if data == "ignore" {
		return ""
	}

	checkForUpgrade := false
	if data == "never" || force {
		checkForUpgrade = true
	}

	if !checkForUpgrade {
		configValue := strings.TrimPrefix(strings.TrimSuffix(string(data),"\""), "\"")
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
		setConfigValue("last-upgrade-check", time.Now().Format(time.RFC3339))
		err := saveConfig()
		if err != nil {
			return ""
		}

		latestVersion := getLatestReleaseVersion()
		if versionCompare(VERSION, latestVersion) == 1 {
			if !force {
				fmt.Printf(
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
	latestVersion := path.Base(location)

	return latestVersion
}

func upgradeCli(latestVersion string) bool {
	status := getSpinner("Upgrading Akamai CLI", "Upgrading Akamai CLI...... ["+color.GreenString("OK")+"]\n\n")

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
		return false
	}

	url := buf.String()
	status.Start()
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil || resp.StatusCode != 200 {
		status.FinalMSG = status.Prefix + "...... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		color.Red("Unable to download release, please try again.")
		return false
	}

	shaResp, err := http.Get(url + ".sig")
	defer shaResp.Body.Close()
	if err != nil || shaResp.StatusCode != 200 {
		status.FinalMSG = status.Prefix + "...... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		color.Red("Unable to retrieve signature for verification, please try again.")
		return false
	}

	shabody, err := ioutil.ReadAll(shaResp.Body)
	if err != nil {
		status.FinalMSG = status.Prefix + "...... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		color.Red("Unable to retrieve signature for verification, please try again.")
		return false
	}

	shasum, err := hex.DecodeString(strings.TrimSpace(string(shabody)))
	if err != nil {
		status.FinalMSG = status.Prefix + "...... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		color.Red("Unable to retrieve signature for verification, please try again.")
		return false
	}

	selfPath, err := osext.Executable()
	if err != nil {
		status.FinalMSG = status.Prefix + "...... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		color.Red("Unable to determine install location")
		return false
	}

	err = update.Apply(resp.Body, update.Options{TargetPath: selfPath, Checksum: shasum})
	if err != nil {
		status.FinalMSG = status.Prefix + "...... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		if rerr := update.RollbackError(err); rerr != nil {
			color.Red("Unable to install or rollback, please re-install.")
			os.Exit(1)
			return false
		} else if strings.HasPrefix(err.Error(), "Upgrade file has wrong checksum.") {
			color.Red(err.Error())
			color.Red("Checksums do not match, please try again.")
		}
		return false
	}

	status.Stop()

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
