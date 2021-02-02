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
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/fatih/color"
	"github.com/kardianos/osext"

	"github.com/akamai/cli/pkg/config"

	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
)

const (
	windowsOS = "windows"
)

func firstRun(ctx context.Context) error {
	term := terminal.Get(ctx)
	if !term.IsTTY() {
		return nil
	}

	bannerShown, err := firstRunCheckInPath(ctx)
	if err != nil {
		return err
	}

	bannerShown, err = firstRunCheckUpgrade(ctx, bannerShown)
	if err != nil {
		return err
	}
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
	writablePaths := make([]string, 0)

	var bannerShown bool
	if config.GetConfigValue("cli", "install-in-path") == "no" {
		inPath = true
		bannerShown, err = firstRunCheckUpgrade(ctx, !inPath)
		if err != nil {
			return false, err
		}
	}

	if len(paths) == 0 {
		inPath = true
		bannerShown, err = firstRunCheckUpgrade(ctx, !inPath)
		if err != nil {
			return false, err
		}
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
			bannerShown, err = firstRunCheckUpgrade(ctx, false)
			if err != nil {
				return false, err
			}
		}
	}

	if !inPath && len(writablePaths) > 0 {
		if !bannerShown {
			terminal.ShowBanner(ctx)
			bannerShown = true
		}
		answer, err := term.Confirm("Akamai CLI is not installed in your PATH, would you like to install it? [Y/n]: ", true)
		if err != nil {
			return false, err
		}
		if !answer {
			config.SetConfigValue("cli", "install-in-path", "no")
			if err := config.SaveConfig(ctx); err != nil {
				return false, err
			}
			if _, err = firstRunCheckUpgrade(ctx, true); err != nil {
				return false, err
			}
			return true, nil
		}

		choosePath(ctx, writablePaths, selfPath)
	}

	return bannerShown, nil
}

func choosePath(ctx context.Context, writablePaths []string, selfPath string) {
	term := terminal.Get(ctx)
	term.Writeln(color.YellowString("Choose where you would like to install Akamai CLI:"))
	answer, err := term.Prompt("Choose where you would like to install Akamai CLI:", writablePaths...)
	if err != nil {
		term.Spinner().Start(string(terminal.SpinnerStatusFail))
		term.Writeln(color.RedString(err.Error()))
		return
	}

	var suffix string
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
		return
	}
	term.Spinner().Start(string(terminal.SpinnerStatusOK))
}

func firstRunCheckUpgrade(ctx context.Context, bannerShown bool) (bool, error) {
	term := terminal.Get(ctx)

	if config.GetConfigValue("cli", "last-upgrade-check") == "" {
		if !bannerShown {
			bannerShown = true
			terminal.ShowBanner(ctx)
		}
		answer, err := term.Confirm("Akamai CLI can auto-update itself, would you like to enable daily checks? [Y/n]: ", true)
		if err != nil {
			return false, err
		}
		if !answer {
			config.SetConfigValue("cli", "last-upgrade-check", "ignore")
			if err := config.SaveConfig(ctx); err != nil {
				return false, err
			}
			return bannerShown, nil
		}

		config.SetConfigValue("cli", "last-upgrade-check", "never")
		if err := config.SaveConfig(ctx); err != nil {
			return false, err
		}
	}

	return bannerShown, nil
}
