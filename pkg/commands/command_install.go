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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var (
	thirdPartyDisclaimer = color.CyanString("Disclaimer: You are installing a third-party package, subject to its own terms and conditions. Akamai makes no warranty or representation with respect to the third-party package.")
)

func cmdInstall(git git.Repository, langManager packages.LangManager) cli.ActionFunc {
	return func(c *cli.Context) (e error) {
		start := time.Now()
		c.Context = log.WithCommandContext(c.Context, c.Command.Name)
		logger := log.WithCommand(c.Context, c.Command.Name)
		logger.Debug("INSTALL START")
		defer func() {
			if e == nil {
				logger.Debugf("INSTALL FINISH: %v", time.Since(start))
			} else {
				var exitErr cli.ExitCoder
				if errors.As(e, &exitErr) && exitErr.ExitCode() == 0 {
					logger.Warnf("INSTALL WARN: %v", e.Error())
				} else {
					logger.Errorf("INSTALL ERROR: %v", e.Error())
				}
			}
		}()
		if !c.Args().Present() {
			return cli.Exit(color.RedString("You must specify a repository URL"), 1)
		}

		oldCmds := getCommands(c)

		for _, repo := range c.Args().Slice() {
			repo = tools.Githubize(repo)
			subCmd, err := installPackage(c.Context, git, langManager, repo, c.Bool("force"))
			if err != nil {
				// Only track public github repos
				if isPublicRepo(repo) {
					stats.TrackEvent(c.Context, "package.install", "failed", repo)
				}
				return err
			}
			c.App.Commands = append(c.App.Commands, subcommandToCliCommands(*subCmd, git, langManager)...)
			sortCommands(c.App.Commands)

			if isPublicRepo(repo) {
				stats.TrackEvent(c.Context, "package.install", "success", repo)
			}
		}

		packageListDiff(c, oldCmds)

		return nil
	}
}

func packageListDiff(c *cli.Context, oldcmds []subcommands) {
	cmds := getCommands(c)

	var old []command
	for _, oldcmd := range oldcmds {
		old = append(old, oldcmd.Commands...)
	}

	var newCmds []command
	for _, newcmd := range cmds {
		newCmds = append(newCmds, newcmd.Commands...)
	}

	var added = make(map[string]bool)
	var removed = make(map[string]bool)

	for _, newCmd := range newCmds {
		found := false
		for _, oldCmd := range old {
			if newCmd.Name == oldCmd.Name {
				found = true
				break
			}
		}

		if !found {
			added[newCmd.Name] = true
		}
	}

	for _, oldCmd := range old {
		found := false
		for _, newCmd := range newCmds {
			if newCmd.Name == oldCmd.Name {
				found = true
				break
			}
		}

		if !found {
			removed[oldCmd.Name] = true
		}
	}

	listInstalledCommands(c, added, removed)
}

func isPublicRepo(repo string) bool {
	return !strings.Contains(repo, ":") || strings.HasPrefix(repo, "https://github.com/")
}

func installPackage(ctx context.Context, gitRepo git.Repository, langManager packages.LangManager, repo string, forceBinary bool) (*subcommands, error) {
	logger := log.FromContext(ctx)
	srcPath, err := tools.GetAkamaiCliSrcPath()
	if err != nil {
		return nil, err
	}

	term := terminal.Get(ctx)

	spin := term.Spinner()

	spin.Start("Attempting to fetch command from %s...", repo)

	dirName := strings.TrimSuffix(filepath.Base(repo), ".git")
	packageDir := filepath.Join(srcPath, dirName)
	if _, err = os.Stat(packageDir); err == nil {
		spin.Stop(terminal.SpinnerStatusWarn)
		warningMsg := fmt.Sprintf("Package directory already exists (%s). To reinstall this package, first run 'akamai uninstall' command.", packageDir)
		return nil, cli.Exit(color.YellowString(warningMsg), 0)
	}

	err = gitRepo.Clone(ctx, packageDir, repo, false, spin)
	if err != nil {
		if err := os.RemoveAll(packageDir); err != nil {
			return nil, err
		}
		spin.Stop(terminal.SpinnerStatusFail)

		errorMsg := "Unable to clone repository: " + err.Error()
		logger.Error(errorMsg)
		return nil, cli.Exit(color.RedString(errorMsg), 1)
	}
	spin.OK()

	if !strings.HasPrefix(repo, "https://github.com/akamai/cli-") && !strings.HasPrefix(repo, "git@github.com:akamai/cli-") {
		term.Printf(color.CyanString(thirdPartyDisclaimer))
	}

	ok, subCmd := installPackageDependencies(ctx, langManager, packageDir, forceBinary, logger)
	if !ok {
		if err := os.RemoveAll(packageDir); err != nil {
			return nil, err
		}
		return nil, cli.Exit("Unable to install selected package", 1)
	}

	return subCmd, nil
}

func installPackageDependencies(ctx context.Context, langManager packages.LangManager, dir string, forceBinary bool, logger log.Logger) (bool, *subcommands) {
	cmdPackage, err := readPackage(dir)

	term := terminal.Get(ctx)

	term.Spinner().Start("Installing...")
	if err != nil {
		term.Spinner().Stop(terminal.SpinnerStatusFail)
		term.Writeln(err.Error())
		logger.Error(err.Error())
		return false, nil
	}

	var commands []string
	for _, cmd := range cmdPackage.Commands {
		commands = append(commands, cmd.Name)
	}

	err = langManager.Install(ctx, dir, cmdPackage.Requirements, commands)
	if errors.Is(err, packages.ErrUnknownLang) {
		term.Spinner().WarnOK()
		warnMsg := "Package installed successfully, however package type is unknown, and may or may not function correctly."
		term.Writeln(color.CyanString(warnMsg))
		logger.Warn(warnMsg)
		return true, &cmdPackage
	}

	if err == nil {
		term.Spinner().OK()
		return true, &cmdPackage
	}

	first := true
	for _, cmd := range cmdPackage.Commands {
		if cmd.Bin != "" {
			if first {
				first = false
				term.Spinner().Stop(terminal.SpinnerStatusWarn)
				term.Writeln(color.CyanString(err.Error()))
				logger.Warn(err.Error())
				if !forceBinary {
					if !term.IsTTY() {
						return false, nil
					}

					answer, err := term.Confirm("Binary command(s) found, would you like to download and install it?", true)
					if err != nil {
						term.WriteError(err.Error())
						logger.Error(err.Error())
						return false, nil
					}

					if !answer {
						return false, nil
					}
				}

				if err := os.MkdirAll(filepath.Join(dir, "bin"), 0700); err != nil {
					return false, nil
				}

				term.Spinner().Start("Downloading binary...")
			}

			if err = downloadBin(ctx, filepath.Join(dir, "bin"), cmd); err != nil {
				term.Spinner().Stop(terminal.SpinnerStatusFail)
				errorMsg := "Unable to download binary: " + err.Error()
				term.Writeln(color.RedString(errorMsg))
				logger.Error(errorMsg)
				return false, nil
			}
		}

		if first {
			term.Spinner().Stop(terminal.SpinnerStatusFail)
			term.Writeln(color.RedString(err.Error()))
			logger.Error(err.Error())
			return false, nil
		}
	}

	term.Spinner().Stop(terminal.SpinnerStatusOK)

	return true, &cmdPackage
}
