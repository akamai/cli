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
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/packages"
	"github.com/urfave/cli/v2"
	"io"
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
	cfg := config.Get(ctx)

	if !term.IsTTY() {
		return ""
	}

	data, _ := cfg.GetValue("cli", "last-upgrade-check")
	data = strings.TrimSpace(data)

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
		cfg.SetValue("cli", "last-upgrade-check", time.Now().Format(time.RFC3339))
		err := cfg.Save(ctx)
		if err != nil {
			return ""
		}

		latestVersion := getLatestReleaseVersion(ctx)
		comp := version.Compare(version.Version, latestVersion)
		if comp == 1 {
			term.Spinner().Stop(terminal.SpinnerStatusOK)
			answer, err := term.Confirm(fmt.Sprintf(
				"New upgrade found: %s (you are running: %s). Upgrade now? [Y/n]: ",
				color.BlueString(latestVersion),
				color.BlueString(version.Version),
			), true)
			if err != nil {
				return ""
			}

			if !answer {
				return ""
			}
			return latestVersion
		}
		if comp == 0 {
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
func UpgradeCli(ctx context.Context, latestVersion string, langManager packages.LangManager) bool {
	term := terminal.Get(ctx)
	logger := log.FromContext(ctx)

	term.Spinner().Start("Upgrading Akamai CLI")

	repo := "https://github.com/akamai/cli"
	if r := os.Getenv("CLI_REPOSITORY"); r != "" {
		repo = r
	}
	cmd := command{
		Version: latestVersion,
		Bin:     fmt.Sprintf("%s/archive/{{.Version}}.zip", repo),
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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error(err.Error())
		}
	}()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	selfPath, err := osext.Executable()
	if err != nil {
		term.Spinner().Fail()
		term.Writeln(color.RedString("Unable to determine install location"))
		return false
	}
	if err = unzipAndUpgrade(ctx, langManager, content, cmd.Version, selfPath); err != nil {
		term.Spinner().Fail()
		term.Writeln(color.RedString("Error occurred while performing upgrade:"))
		term.Writeln(color.RedString(err.Error()))
		return false
	}

	term.Spinner().OK()

	os.Args[0] = selfPath

	if err := passthruCommand(os.Args); err != nil {
		cli.OsExiter(1)
		return false
	}
	cli.OsExiter(0)

	return true
}

func getUpgradeCommand(langManager packages.LangManager) *cli.Command {
	return &cli.Command{
		Name:        "upgrade",
		Description: "Upgrade Akamai CLI to the latest version",
		Action:      cmdUpgrade(langManager),
	}
}

func unzipAndUpgrade(ctx context.Context, langManager packages.LangManager, content []byte, ver, selfPath string) error {
	term := terminal.Get(ctx)
	targetDir := fmt.Sprintf("cli-upgrade-%s", ver)
	if err := os.Mkdir(targetDir, os.ModePerm); err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(targetDir)
	}()
	zipName := fmt.Sprintf("upgrade-cli-%s.zip", ver)
	f, err := os.Create(zipName)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		_ = os.RemoveAll(zipName)
	}()

	if _, err = f.Write(content); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}

	if _, err = unzip(zipName, targetDir); err != nil {
		return err
	}

	dirName := fmt.Sprintf("%s/cli-%s", targetDir, ver)
	mainPath := "main"
	// if version is greater than 1.1.5, we have to build from different directory
	if comp := version.Compare(ver, "1.1.5"); comp == -1 {
		mainPath = "cli/main.go"
	}
	if err = langManager.Install(ctx, dirName, packages.LanguageRequirements{Go: runtime.Version()}, []string{mainPath}); err != nil {
		return err
	}
	reader, err := os.Open(fmt.Sprintf("%s/akamai-main", dirName))
	if err != nil {
		return err
	}

	err = update.Apply(reader, update.Options{TargetPath: selfPath})
	if err != nil {
		if rerr := update.RollbackError(err); rerr != nil {
			term.Writeln(color.RedString("Unable to install or rollback, please re-install."))
			if err = os.RemoveAll(targetDir); err != nil {
				return err
			}
			if err = os.RemoveAll(zipName); err != nil {
				return err
			}
			os.Exit(1)
			return err
		}
		return err
	}
	return nil
}

func unzip(src string, dest string) ([]string, error) {
	var filenames []string
	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}
		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(fpath, os.ModePerm); err != nil {
				return nil, err
			}
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
