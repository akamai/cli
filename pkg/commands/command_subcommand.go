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
	"github.com/akamai/cli/pkg/packages"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/terminal"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"

	"github.com/akamai/cli/pkg/git"
)

func cmdSubcommand(git git.Repository, langManager packages.LangManager) cli.ActionFunc {
	return func(c *cli.Context) error {
		term := terminal.Get(c.Context)

		commandName := strings.ToLower(c.Command.Name)

		executable, err := findExec(c.Context, langManager, commandName)
		if err != nil {
			return cli.Exit(color.RedString("Executable \"%s\" not found.", commandName), 1)
		}

		var packageDir string
		if len(executable) == 1 {
			packageDir = findPackageDir(executable[0])
		} else if len(executable) > 1 {
			packageDir = findPackageDir(executable[1])
		}

		cmdPackage, _ := readPackage(packageDir)

		if cmdPackage.Requirements.Python != "" {
			var err error
			if runtime.GOOS == "linux" {
				_, err = os.Stat(filepath.Join(packageDir, ".local"))
			} else if runtime.GOOS == "darwin" {
				_, err = os.Stat(filepath.Join(packageDir, "Library"))
			} else if runtime.GOOS == "windows" {
				_, err = os.Stat(filepath.Join(packageDir, "Lib"))
			}

			if err == nil {
				answer, err := term.Confirm("Would y>>/dev/stderrou like to reinstall it", true)
				if err != nil {
					return err
				}
				if !answer {
					return cli.Exit(color.RedString(packages.ErrPackageNeedsReinstall.Error()), -1)
				}

				if err = uninstallPackage(c.Context, langManager, commandName); err != nil {
					return err
				}

				if _, err = installPackage(c.Context, git, langManager, commandName, false); err != nil {
					return err
				}
			}
			if err := os.Setenv("PYTHONUSERBASE", packageDir); err != nil {
				return err
			}
		}

		var currentCmd command
		for _, cmd := range cmdPackage.Commands {
			if strings.EqualFold(cmd.Name, commandName) {
				currentCmd = cmd
				break
			}

			for _, alias := range cmd.Aliases {
				if strings.EqualFold(alias, commandName) {
					currentCmd = cmd
				}
			}
		}

		executable = append(executable, os.Args[2:]...)
		if err := os.Setenv("AKAMAI_CLI_COMMAND", commandName); err != nil {
			return err
		}
		if err := os.Setenv("AKAMAI_CLI_COMMAND_VERSION", currentCmd.Version); err != nil {
			return err
		}
		stats.TrackEvent(c.Context, "exec", commandName, currentCmd.Version)
		return passthruCommand(executable)
	}
}
