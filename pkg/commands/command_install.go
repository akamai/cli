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
	"os"
	"path/filepath"
	"strings"

	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
	"gopkg.in/src-d/go-git.v4"

	"github.com/fatih/color"
)

var (
	// ThirdPartyDisclaimer is the message to be used when third party packages are installed
	ThirdPartyDisclaimer = color.CyanString("Disclaimer: You are installing a third-party package, subject to its own terms and conditions. Akamai makes no warranty or representation with respect to the third-party package.")
)

func cmdInstall(c *cli.Context) error {
	if !c.Args().Present() {
		return cli.Exit(color.RedString("You must specify a repository URL"), 1)
	}

	oldCmds := getCommands()

	for _, repo := range c.Args().Slice() {
		repo = tools.Githubize(repo)
		err := installPackage(c.Context, repo, c.Bool("force"))
		if err != nil {
			// Only track public github repos
			if isPublicRepo(repo) {
				stats.TrackEvent(c.Context, "package.install", "failed", repo)
			}
			return err
		}

		if isPublicRepo(repo) {
			stats.TrackEvent(c.Context, "package.install", "success", repo)
		}
	}

	packageListDiff(c.Context, oldCmds)

	return nil
}

func packageListDiff(ctx context.Context, oldcmds []subcommands) {
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

	listInstalledCommands(ctx, added, removed)
}

func isPublicRepo(repo string) bool {
	return !strings.Contains(repo, ":") || strings.HasPrefix(repo, "https://github.com/")
}

func installPackage(ctx context.Context, repo string, forceBinary bool) error {
	srcPath, err := tools.GetAkamaiCliSrcPath()
	if err != nil {
		return err
	}

	term := terminal.Get(ctx)

	spin := term.Spinner()

	spin.Start("Attempting to fetch command from %s...", repo)

	dirName := strings.TrimSuffix(filepath.Base(repo), ".git")
	packageDir := filepath.Join(srcPath, dirName)
	if _, err = os.Stat(packageDir); err == nil {
		spin.Stop(terminal.SpinnerStatusFail)
		return cli.NewExitError(color.RedString("Package directory already exists (%s)", packageDir), 1)
	}

	_, err = git.PlainClone(packageDir, false, &git.CloneOptions{
		URL:      repo,
		Progress: spin,
		Depth:    1,
	})

	if err != nil {
		if err := os.RemoveAll(packageDir); err != nil {
			return err
		}
		spin.Stop(terminal.SpinnerStatusFail)

		os.RemoveAll(packageDir)

		return cli.NewExitError(color.RedString("Unable to clone repository: "+err.Error()), 1)
	}

	if strings.HasPrefix(repo, "https://github.com/akamai/cli-") != true && strings.HasPrefix(repo, "git@github.com:akamai/cli-") != true {
		term.Printf(color.CyanString("Disclaimer: You are installing a third-party package, subject to its own terms and conditions. Akamai makes no warranty or representation with respect to the third-party package."))
	}

	if !installPackageDependencies(ctx, packageDir, forceBinary) {
		os.RemoveAll(packageDir)
		return cli.NewExitError("", 1)
	}

	return nil
}

func installPackageDependencies(ctx context.Context, dir string, forceBinary bool) bool {
	logger := log.FromContext(ctx)

	cmdPackage, err := readPackage(dir)

	term := terminal.Get(ctx)

	term.Spinner().Start("Installing...")
	if err != nil {
		term.Spinner().Stop(terminal.SpinnerStatusFail)
		term.Writeln(err.Error())
		return false
	}

	lang := determineCommandLanguage(cmdPackage)

	switch lang {
	case languagePHP:
		err = packages.InstallPHP(ctx, dir, cmdPackage.Requirements.Php)
	case languageJavaScript:
		err = packages.InstallJavaScript(ctx, dir, cmdPackage.Requirements.Node)
	case languageRuby:
		err = packages.InstallRuby(ctx, dir, cmdPackage.Requirements.Ruby)
	case languagePython:
		err = packages.InstallPython(ctx, dir, cmdPackage.Requirements.Python)
	case languageGO:
		var commands []string
		for _, cmd := range cmdPackage.Commands {
			commands = append(commands, cmd.Name)
		}
		err = packages.InstallGolang(logger, dir, cmdPackage.Requirements.Go, commands)
	default:
		term.Spinner().Stop(terminal.SpinnerStatusWarnOK)
		term.Writeln(color.CyanString("Package installed successfully, however package type is unknown, and may or may not function correctly."))
		return true
	}

	if err == nil {
		term.Spinner().OK()
		return true
	}

	first := true
	for _, cmd := range cmdPackage.Commands {
		if cmd.Bin != "" {
			if first {
				first = false
				term.Spinner().Stop(terminal.SpinnerStatusWarn)
				term.Writeln(color.CyanString(err.Error()))
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

				term.Spinner().Start("Downloading binary...")
			}

			if !downloadBin(ctx, filepath.Join(dir, "bin"), cmd) {
				term.Spinner().Stop(terminal.SpinnerStatusFail)
				term.Writeln(color.RedString("Unable to download binary: " + err.Error()))
				return false
			}
		}

		if first {
			first = false
			term.Spinner().Stop(terminal.SpinnerStatusFail)
			term.Writeln(color.RedString(err.Error()))
			return false
		}
	}

	term.Spinner().Stop(terminal.SpinnerStatusOK)
	return true
}
