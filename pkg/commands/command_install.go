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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/akamai/cli/pkg/app"
	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
)

var (
	// ThirdPartyDisclaimer is the message to be used when third party packages are installed
	ThirdPartyDisclaimer = color.CyanString("Disclaimer: You are installing a third-party package, subject to its own terms and conditions. Akamai makes no warranty or representation with respect to the third-party package.")
)

func cmdInstall(git git.Repository) cli.ActionFunc {
	return func(c *cli.Context) error {
		logger := log.WithCommand(c.Context, c.Command.Name)
		if !c.Args().Present() {
			return cli.Exit(color.RedString("You must specify a repository URL"), 1)
		}

		oldCmds := getCommands()

		for _, repo := range c.Args().Slice() {
			repo = tools.Githubize(repo)
			err := installPackage(c.Context, git, logger, repo, c.Bool("force"))
			if err != nil {
				// Only track public github repos
				if isPublicRepo(repo) {
					stats.TrackEvent("package.install", "failed", repo)
				}
				return err
			}

			if isPublicRepo(repo) {
				stats.TrackEvent("package.install", "success", repo)
			}
		}

		packageListDiff(oldCmds)

		return nil
	}
}

func packageListDiff(oldcmds []subcommands) {
	cmds := getCommands()

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

	listInstalledCommands(added, removed)
}

func isPublicRepo(repo string) bool {
	return !strings.Contains(repo, ":") || strings.HasPrefix(repo, "https://github.com/")
}

func installPackage(ctx context.Context, git git.Repository, logger log.Logger, repo string, forceBinary bool) error {
	srcPath, err := tools.GetAkamaiCliSrcPath()
	if err != nil {
		return err
	}

	term := terminal.Standard()

	spin := term.Spinner()

	spin.Start("Attempting to fetch command from %s", repo)

	dirName := strings.TrimSuffix(filepath.Base(repo), ".git")
	packageDir := filepath.Join(srcPath, dirName)
	if _, err = os.Stat(packageDir); err == nil {
		spin.Stop(terminal.SpinnerStatusFail)
		return cli.NewExitError(color.RedString("Package directory already exists (%s)", packageDir), 1)
	}

	_, err = git.Clone(ctx, packageDir, repo, false, spin, 1)
	if err != nil {
		if err := os.RemoveAll(packageDir); err != nil {
			return err
		}
		spin.Stop(terminal.SpinnerStatusFail)

		return cli.Exit(color.RedString("Unable to clone repository: "+err.Error()), 1)
	}

	spin.Stop(terminal.SpinnerStatusOK)

	if !strings.HasPrefix(repo, "https://github.com/akamai/cli-") && !strings.HasPrefix(repo, "git@github.com:akamai/cli-") {
		term.Writef(ThirdPartyDisclaimer)
	}

	if !installPackageDependencies(logger, packageDir, forceBinary) {
		if err := os.RemoveAll(packageDir); err != nil {
			return err
		}
		return cli.Exit("", 1)
	}

	return nil
}

func installPackageDependencies(logger log.Logger, dir string, forceBinary bool) bool {
	term := terminal.Standard()

	spin := term.Spinner()

	spin.Start("Installing...")

	cmdPackage, err := readPackage(dir)

	if err != nil {
		spin.Stop(terminal.SpinnerStatusFail)
		fmt.Fprintln(app.App.Writer, err.Error())
		return false
	}

	lang := determineCommandLanguage(cmdPackage)

	switch lang {
	case languagePHP:
		err = packages.InstallPHP(logger, dir, cmdPackage.Requirements.Php)
	case languageJavaScript:
		err = packages.InstallJavaScript(logger, dir, cmdPackage.Requirements.Node)
	case languageRuby:
		err = packages.InstallRuby(logger, dir, cmdPackage.Requirements.Ruby)
	case languagePython:
		err = packages.InstallPython(logger, dir, cmdPackage.Requirements.Python)
	case languageGO:
		var commands []string
		for _, cmd := range cmdPackage.Commands {
			commands = append(commands, cmd.Name)
		}
		err = packages.InstallGolang(logger, dir, cmdPackage.Requirements.Go, commands)
	default:
		spin.Stop(terminal.SpinnerStatusWarnOK)

		term.Writef(color.CyanString("Package installed successfully, however package type is unknown, and may or may not function correctly."))

		return true
	}

	if err == nil {
		spin.Stop(terminal.SpinnerStatusOK)
		return true
	}

	first := true
	for _, cmd := range cmdPackage.Commands {
		if cmd.Bin != "" {
			if !first {
				continue
			}
			first = false
			spin.Stop(terminal.SpinnerStatusWarnOK)
			term.Writef(color.CyanString(err.Error()))
			if !forceBinary {
				if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
					return false
				}

				answer, _ := term.Confirm("Binary command(s) found, would you like to download and install it?", true)

				if !answer {
					return false
				}
			}

			if err := os.MkdirAll(filepath.Join(dir, "bin"), 0700); err != nil {
				return false
			}

			spin.Start("Downloading binary...")

			if !downloadBin(logger, filepath.Join(dir, "bin"), cmd) {
				spin.Stop(terminal.SpinnerStatusFail)

				term.Writef(color.RedString("Unable to download binary: " + err.Error()))
				return false
			}
		}

		if first {
			first = false

			spin.Stop(terminal.SpinnerStatusFail)

			term.Writef(color.RedString(err.Error()))

			return first
		}
	}

	spin.Stop(terminal.SpinnerStatusOK)

	return true
}
