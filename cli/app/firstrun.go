//+build !nofirstrun

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

package app

import (
	"fmt"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/tools"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	akamai "github.com/akamai/cli-common-golang"
	"github.com/fatih/color"
	"github.com/kardianos/osext"
	"github.com/mattn/go-isatty"
)

func firstRun() error {
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return nil
	}

	bannerShown, err := firstRunCheckInPath()
	if err != nil {
		return err
	}

	bannerShown = firstRunCheckUpgrade(bannerShown)
	stats.FirstRunCheckStats(bannerShown)

	return nil
}

func firstRunCheckInPath() (bool, error) {
	selfPath, err := osext.Executable()
	if err != nil {
		return false, err
	}
	os.Args[0] = selfPath
	dirPath := filepath.Dir(selfPath)

	if runtime.GOOS == "windows" {
		dirPath = strings.ToLower(dirPath)
	}

	sysPath := os.Getenv("PATH")
	paths := filepath.SplitList(sysPath)
	inPath := false
	writablePaths := []string{}

	var bannerShown bool
	if config.GetConfigValue("cli", "install-in-path") == "no" {
		inPath = true
		bannerShown = firstRunCheckUpgrade(!inPath)
	}

	if len(paths) == 0 {
		inPath = true
		bannerShown = firstRunCheckUpgrade(!inPath)
	}

	for _, path := range paths {
		if len(strings.TrimSpace(path)) == 0 {
			continue
		}

		if runtime.GOOS == "windows" {
			path = strings.ToLower(path)
		}

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			if err := checkAccess(path, ACCESS_W_OK); err == nil {
				writablePaths = append(writablePaths, path)
			}
		}

		if path == dirPath {
			inPath = true
			bannerShown = firstRunCheckUpgrade(false)
		}
	}

	if !inPath && len(writablePaths) > 0 {
		if !bannerShown {
			tools.ShowBanner()
			bannerShown = true
		}
		fmt.Fprint(akamai.App.Writer, "Akamai CLI is not installed in your PATH, would you like to install it? [Y/n]: ")
		answer := ""
		fmt.Scanln(&answer)
		if answer != "" && strings.ToLower(answer) != "y" {
			config.SetConfigValue("cli", "install-in-path", "no")
			config.SaveConfig()
			firstRunCheckUpgrade(true)
			return true, nil
		}

		choosePath(writablePaths, answer, selfPath)
	}

	return bannerShown, nil
}

func choosePath(writablePaths []string, answer string, selfPath string) {
	fmt.Fprintln(akamai.App.Writer, color.YellowString("Choose where you would like to install Akamai CLI:"))
	for i, path := range writablePaths {
		fmt.Fprintf(akamai.App.Writer, "(%d) %s\n", i+1, path)
	}
	fmt.Fprint(akamai.App.Writer, "Enter a number: ")
	answer = ""
	fmt.Scanln(&answer)
	index, err := strconv.Atoi(answer)
	if err != nil {
		fmt.Fprintln(akamai.App.Writer, color.RedString("Invalid choice, try again"))
		choosePath(writablePaths, answer, selfPath)
	}
	if answer == "" || index < 1 || index > len(writablePaths) {
		fmt.Fprintln(akamai.App.Writer, color.RedString("Invalid choice, try again"))
		choosePath(writablePaths, answer, selfPath)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	newPath := filepath.Join(writablePaths[index-1], "akamai"+suffix)
	akamai.StartSpinner(
		"Installing to "+newPath+"...",
		"Installing to "+newPath+"...... ["+color.GreenString("OK")+"]\n",
	)
	err = tools.MoveFile(selfPath, newPath)

	os.Args[0] = newPath
	if err != nil {
		akamai.StopSpinnerFail()
		fmt.Fprintln(akamai.App.Writer, color.RedString(err.Error()))
	}
	akamai.StopSpinnerOk()
}

func firstRunCheckUpgrade(bannerShown bool) bool {
	if config.GetConfigValue("cli", "last-upgrade-check") == "" {
		if !bannerShown {
			bannerShown = true
			tools.ShowBanner()
		}
		fmt.Fprint(akamai.App.Writer, "Akamai CLI can auto-update itself, would you like to enable daily checks? [Y/n]: ")

		answer := ""
		fmt.Scanln(&answer)
		if answer != "" && strings.ToLower(answer) != "y" {
			config.SetConfigValue("cli", "last-upgrade-check", "ignore")
			config.SaveConfig()
			return bannerShown
		}

		config.SetConfigValue("cli", "last-upgrade-check", "never")
		config.SaveConfig()
	}

	return bannerShown
}
