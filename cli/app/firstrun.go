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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/fatih/color"
	"github.com/kardianos/osext"
	"github.com/mattn/go-isatty"

	"github.com/akamai/cli/pkg/config"

	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
)

const (
	windowsOS = "windows"
)

func firstRun(ctx context.Context) error {
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return nil
	}

	bannerShown, err := firstRunCheckInPath(ctx)
	if err != nil {
		return err
	}

	bannerShown = firstRunCheckUpgrade(ctx, bannerShown)
	stats.FirstRunCheckStats(ctx, bannerShown)

	return nil
}

func firstRunCheckInPath(ctx context.Context) (bool, error) {
	term := terminal.Get(ctx)

	selfPath, err := osext.Executable()
	if err != nil {
		return false, err
	}
	os.Args[0] = selfPath
	dirPath := filepath.Dir(selfPath)

	if runtime.GOOS == windowsOS {
		dirPath = strings.ToLower(dirPath)
	}

	sysPath := os.Getenv("PATH")
	paths := filepath.SplitList(sysPath)
	inPath := false
	writablePaths := []string{}

	var bannerShown bool
	if config.GetConfigValue("cli", "install-in-path") == "no" {
		inPath = true
		bannerShown = firstRunCheckUpgrade(ctx, !inPath)
	}

	if len(paths) == 0 {
		inPath = true
		bannerShown = firstRunCheckUpgrade(ctx, !inPath)
	}

	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}

		if runtime.GOOS == windowsOS {
			path = strings.ToLower(path)
		}

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			if err := checkAccess(path, unix.W_OK); err == nil {
				writablePaths = append(writablePaths, path)
			}
		}

		if path == dirPath {
			inPath = true
			bannerShown = firstRunCheckUpgrade(ctx, false)
		}
	}

	if !inPath && len(writablePaths) > 0 {
		if !bannerShown {
			terminal.ShowBanner(ctx)
			bannerShown = true
		}
		term.Printf("Akamai CLI is not installed in your PATH, would you like to install it? [Y/n]: ")
		answer := ""
		fmt.Scanln(&answer)
		if answer != "" && strings.EqualFold(answer, "y") {
			config.SetConfigValue("cli", "install-in-path", "no")
			config.SaveConfig(ctx)
			firstRunCheckUpgrade(ctx, true)
			return true, nil
		}

		choosePath(ctx, writablePaths, answer, selfPath)
	}

	return bannerShown, nil
}

func choosePath(ctx context.Context, writablePaths []string, answer, selfPath string) {
	term := terminal.Get(ctx)
	term.Writeln(color.YellowString("Choose where you would like to install Akamai CLI:"))
	answer, err := term.Prompt("Choose where you would like to install Akamai CLI:", writablePaths...)
	if err != nil {
		panic(err)
	}

	suffix := ""
	if runtime.GOOS == windowsOS {
		suffix = ".exe"
	}
	newPath := filepath.Join(answer, "akamai"+suffix)
	term.Spinner().Start("Installing to " + newPath + "...")

	err = tools.MoveFile(selfPath, newPath)

	os.Args[0] = newPath
	if err != nil {
		term.Spinner().Start(string(terminal.SpinnerStatusFail))
		term.Writeln(color.RedString(err.Error()))
	}
	term.Spinner().Start(string(terminal.SpinnerStatusOK))
}

func firstRunCheckUpgrade(ctx context.Context, bannerShown bool) bool {
	term := terminal.Get(ctx)

	if config.GetConfigValue("cli", "last-upgrade-check") == "" {
		if !bannerShown {
			bannerShown = true
			terminal.ShowBanner(ctx)
		}
		term.Printf("Akamai CLI can auto-update itself, would you like to enable daily checks? [Y/n]: ")

		answer := ""
		fmt.Scanln(&answer)
		if answer != "" && strings.EqualFold(answer, "y") {
			config.SetConfigValue("cli", "last-upgrade-check", "ignore")
			config.SaveConfig(ctx)
			return bannerShown
		}

		config.SetConfigValue("cli", "last-upgrade-check", "never")
		config.SaveConfig(ctx)
	}

	return bannerShown
}
