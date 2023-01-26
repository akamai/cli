//go:build !nofirstrun
// +build !nofirstrun

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

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/fatih/color"
	"github.com/kardianos/osext"
)

const (
	windowsOS = "windows"
)

func firstRun(ctx context.Context) error {
	term := terminal.Get(ctx)
	cfg := config.Get(ctx)
	if !term.IsTTY() {
		return nil
	}

	bannerShown, err := firstRunCheckInPath(ctx)
	if err != nil {
		return err
	}

	_, err = firstRunCheckUpgrade(ctx, cfg, bannerShown)
	if err != nil {
		return err
	}

	return nil
}

func firstRunCheckInPath(ctx context.Context) (bool, error) {
	term := terminal.Get(ctx)
	cfg := config.Get(ctx)

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
	if val, _ := cfg.GetValue("cli", "install-in-path"); val == "no" {
		inPath = true
		bannerShown, err = firstRunCheckUpgrade(ctx, cfg, !inPath)
		if err != nil {
			return false, err
		}
	}

	if len(paths) == 0 {
		inPath = true
		bannerShown, err = firstRunCheckUpgrade(ctx, cfg, !inPath)
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
			if err := checkWriteAccess(path); err == nil {
				writablePaths = append(writablePaths, path)
			}
		}

		if path == dirPath {
			inPath = true
			bannerShown, err = firstRunCheckUpgrade(ctx, cfg, false)
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
			cfg.SetValue("cli", "install-in-path", "no")
			if err := cfg.Save(ctx); err != nil {
				return false, err
			}
			if _, err = firstRunCheckUpgrade(ctx, cfg, true); err != nil {
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

	if _, err := term.Writeln(color.YellowString("Choose where you would like to install Akamai CLI:")); err != nil {
		term.WriteError(err.Error())
		return
	}
	answer, err := term.Prompt("Choose where you would like to install Akamai CLI:", writablePaths...)
	if err != nil {
		term.Spinner().Start(string(terminal.SpinnerStatusFail))
		if _, err := term.Writeln(color.RedString(err.Error())); err != nil {
			term.WriteError(err.Error())
		}
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
		term.Spinner().Fail()
		if _, err := term.Writeln(color.RedString(err.Error())); err != nil {
			term.WriteError(err.Error())
		}
		return
	}
	term.Spinner().OK()
}

func firstRunCheckUpgrade(ctx context.Context, cfg config.Config, bannerShown bool) (bool, error) {
	term := terminal.Get(ctx)
	_, ok := cfg.GetValue("cli", "last-upgrade-check")
	if ok {
		return bannerShown, nil
	}
	if !bannerShown {
		bannerShown = true
		terminal.ShowBanner(ctx)
	}
	answer, err := term.Confirm("Akamai CLI can auto-update itself, would you like to enable daily checks? [Y/n]: ", true)
	if err != nil {
		return false, err
	}
	if !answer {
		cfg.SetValue("cli", "last-upgrade-check", "ignore")
		if err := cfg.Save(ctx); err != nil {
			return false, err
		}
		return bannerShown, nil
	}

	cfg.SetValue("cli", "last-upgrade-check", "never")
	if err := cfg.Save(ctx); err != nil {
		return false, err
	}

	return bannerShown, nil
}
