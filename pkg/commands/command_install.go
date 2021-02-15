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
	"github.com/akamai/cli/pkg/log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"

	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
)

var (
	thirdPartyDisclaimer = color.CyanString("Disclaimer: You are installing a third-party package, subject to its own terms and conditions. Akamai makes no warranty or representation with respect to the third-party package.")
)

func cmdInstall(git git.Repository, langManager packages.LangManager) cli.ActionFunc {
	return func(c *cli.Context) error {
		c.Context = log.WithCommandContext(c.Context, c.Command.Name)
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
		for _, cmd := range oldcmd.Commands {
			old = append(old, cmd)
		}
	}

	var newCmds []command
	for _, newcmd := range cmds {
		for _, cmd := range newcmd.Commands {
			newCmds = append(newCmds, cmd)
		}
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
		spin.Stop(terminal.SpinnerStatusFail)
		return nil, cli.Exit(color.RedString("Package directory already exists (%s)", packageDir), 1)
	}

	err = gitRepo.Clone(ctx, packageDir, repo, false, spin, 1)
	if err != nil {
		if err := os.RemoveAll(packageDir); err != nil {
			return nil, err
		}
		spin.Stop(terminal.SpinnerStatusFail)

		return nil, cli.Exit(color.RedString("Unable to clone repository: "+err.Error()), 1)
	}
	spin.OK()

	if !strings.HasPrefix(repo, "https://github.com/akamai/cli-") && !strings.HasPrefix(repo, "git@github.com:akamai/cli-") {
		term.Printf(color.CyanString(thirdPartyDisclaimer))
	}

	ok, subCmd := installPackageDependencies(ctx, langManager, packageDir, forceBinary)
	if !ok {
		if err := os.RemoveAll(packageDir); err != nil {
			return nil, err
		}
		return nil, cli.Exit("Unable to install selected package", 1)
	}

	return subCmd, nil
}

func installPackageDependencies(ctx context.Context, langManager packages.LangManager, dir string, forceBinary bool) (bool, *subcommands) {
	cmdPackage, err := readPackage(dir)

	term := terminal.Get(ctx)

	term.Spinner().Start("Installing...")
	if err != nil {
		term.Spinner().Stop(terminal.SpinnerStatusFail)
		term.Writeln(err.Error())
		return false, nil
	}

	var commands []string
	for _, cmd := range cmdPackage.Commands {
		commands = append(commands, cmd.Name)
	}

	err = langManager.Install(ctx, dir, cmdPackage.Requirements, commands)
	if errors.Is(err, packages.ErrUnknownLang) {
		term.Spinner().WarnOK()
		term.Writeln(color.CyanString("Package installed successfully, however package type is unknown, and may or may not function correctly."))
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
				if !forceBinary {
					if !term.IsTTY() {
						return false, nil
					}

					answer, err := term.Confirm("Binary command(s) found, would you like to download and install it?", true)
					if err != nil {
						term.WriteError(err.Error())
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

			if !downloadBin(ctx, filepath.Join(dir, "bin"), cmd) {
				term.Spinner().Stop(terminal.SpinnerStatusFail)
				term.Writeln(color.RedString("Unable to download binary: " + err.Error()))
				return false, nil
			}
		}

		if first {
			term.Spinner().Stop(terminal.SpinnerStatusFail)
			term.Writeln(color.RedString(err.Error()))
			return false, nil
		}
	}

	term.Spinner().Stop(terminal.SpinnerStatusOK)

	return true, &cmdPackage
}
