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
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/git"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/packages"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/tools"
	"github.com/urfave/cli/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	thirdPartyDisclaimer = color.CyanString("Disclaimer: You are installing a third-party package, subject to its own terms and conditions. Akamai makes no warranty or representation with respect to the third-party package.")
	githubRawURLTemplate = "https://raw.githubusercontent.com/akamai/%s/master/cli.json"
)

func cmdInstall(git git.Repository, langManager packages.LangManager) cli.ActionFunc {
	return func(c *cli.Context) (e error) {
		start := time.Now()
		c.Context = log.WithCommandContext(c.Context, c.Command.Name)
		logger := log.FromContext(c.Context)
		logger.Debug("INSTALL START")
		defer func() {
			if e == nil {
				logger.Debug(fmt.Sprintf("INSTALL FINISH: %v", time.Since(start)))
			} else {
				var exitErr cli.ExitCoder
				if errors.As(e, &exitErr) && exitErr.ExitCode() == 0 {
					logger.Warn(fmt.Sprintf("INSTALL WARN: %v", e.Error()))
				} else {
					logger.Error(fmt.Sprintf("INSTALL ERROR: %v", e.Error()))
				}
			}
		}()
		if !c.Args().Present() {
			return cli.Exit(color.RedString("You must specify a repository URL"), 1)
		}

		oldCmds := getCommands(c)

		for _, repo := range c.Args().Slice() {
			repo = tools.Githubize(repo)
			subCmd, err := installPackage(c.Context, git, langManager, repo)
			if err != nil {

				return err
			}
			c.App.Commands = append(c.App.Commands, subcommandToCliCommands(*subCmd, git, langManager)...)
			sortCommands(c.App.Commands)
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

func installPackage(ctx context.Context, gitRepo git.Repository, langManager packages.LangManager, repo string) (*subcommands, error) {
	logger := log.FromContext(ctx)
	srcPath, err := tools.GetAkamaiCliSrcPath()
	if err != nil {
		return nil, err
	}

	term := terminal.Get(ctx)
	spin := term.Spinner()

	dirName := strings.TrimSuffix(filepath.Base(repo), ".git")
	packageDir := filepath.Join(srcPath, dirName)

	if _, err = os.Stat(packageDir); err == nil {
		warningMsg := fmt.Sprintf("Package directory already exists (%s). To reinstall this package, first run 'akamai uninstall' command.", packageDir)
		return nil, cli.Exit(color.YellowString(warningMsg), 0)
	}

	spin.Start("Attempting to fetch package configuration from %s...", repo)

	base := filepath.Base(dirName)
	url := fmt.Sprintf(githubRawURLTemplate, base)
	cmdPackage, err := readPackageFromGithub(url, dirName)
	if err != nil {
		spin.Stop(terminal.SpinnerStatusFail)
		logger.Error(err.Error())
		if _, err := term.Writeln(err.Error()); err != nil {
			term.WriteError(err.Error())
		}
		if strings.Contains(err.Error(), "404") {
			return nil, cli.Exit(color.RedString(tools.CapitalizeFirstWord(git.ErrPackageNotAvailable.Error())), 1)
		}
		return nil, cli.Exit(color.RedString("Unable to install selected package"), 1)
	}
	spin.OK()

	if isBinary(cmdPackage) {

		ok, subCmd := installPackageBinaries(ctx, packageDir, cmdPackage, logger)
		if ok {
			return subCmd, nil
		}
		// delete package directory
		if err := os.RemoveAll(packageDir); err != nil {
			return nil, err
		}

	}
	spin.Start("Attempting to fetch command from %s...", repo)

	if !strings.HasPrefix(repo, "https://github.com/akamai/cli-") && !strings.HasPrefix(repo, "git@github.com:akamai/cli-") {
		term.Printf(color.CyanString(thirdPartyDisclaimer))
	}
	err = gitRepo.Clone(ctx, packageDir, repo, false, spin)
	if err != nil {
		if err := os.RemoveAll(packageDir); err != nil {
			return nil, err
		}
		spin.Stop(terminal.SpinnerStatusFail)

		logger.Error(cases.Title(language.Und, cases.NoLower).String(err.Error()))
		return nil, cli.Exit(color.RedString(tools.CapitalizeFirstWord(err.Error())), 1)
	}
	spin.OK()

	ok, subCmd := installPackageDependencies(ctx, langManager, packageDir, logger)
	if !ok {
		if err := os.RemoveAll(packageDir); err != nil {
			return nil, err
		}
		return nil, cli.Exit("Unable to install selected package", 1)
	}

	return subCmd, nil
}

func installPackageDependencies(ctx context.Context, langManager packages.LangManager, dir string, logger *slog.Logger) (bool, *subcommands) {
	cmdPackage, err := readPackage(dir)

	term := terminal.Get(ctx)

	term.Spinner().Start("Installing...")
	if err != nil {
		term.Spinner().Stop(terminal.SpinnerStatusFail)
		logger.Error(err.Error())
		if _, err := term.Writeln(err.Error()); err != nil {
			term.WriteError(err.Error())
		}
		return false, nil
	}

	var commands, ldFlags []string
	for _, cmd := range cmdPackage.Commands {
		commands = append(commands, cmd.Name)
		ldFlag := cmd.LdFlags
		if ldFlag != "" {
			ldFlag = fmt.Sprintf(ldFlag, cmd.Version)
		}
		ldFlags = append(ldFlags, ldFlag)
	}

	err = langManager.Install(ctx, dir, cmdPackage.Requirements, commands, ldFlags)
	if errors.Is(err, packages.ErrUnknownLang) {
		term.Spinner().WarnOK()
		warnMsg := "Package installed successfully, however package type is unknown, and may or may not function correctly."
		if _, err := term.Writeln(color.CyanString(warnMsg)); err != nil {
			term.WriteError(err.Error())
			return false, nil
		}
		logger.Warn(warnMsg)
		return true, &cmdPackage
	}

	if err != nil {
		term.Spinner().Stop(terminal.SpinnerStatusFail)
		term.WriteError(err.Error())
		return false, nil

	}

	term.Spinner().OK()
	return true, &cmdPackage

}

func installPackageBinaries(ctx context.Context, dir string, cmdPackage subcommands, logger *slog.Logger) (bool, *subcommands) {

	term := terminal.Get(ctx)
	spin := term.Spinner()
	spin.Start("Installing Binaries...")

	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0700); err != nil {
		return false, nil
	}

	for _, cmd := range cmdPackage.Commands {
		err := downloadBin(ctx, filepath.Join(dir, "bin"), cmd)
		if err != nil {
			warnMsg := fmt.Sprintf("Unable to download binary: %v", err.Error())
			spin.Stop(terminal.SpinnerStatusWarn)
			if _, err := term.Writeln(color.YellowString(warnMsg)); err != nil {
				term.WriteError(err.Error())
				return false, nil
			}
			logger.Warn(warnMsg)

			return false, nil

		}
	}

	err := os.WriteFile(filepath.Join(dir, "cli.json"), cmdPackage.raw, 0644)
	if err != nil {
		spin.Stop(terminal.SpinnerStatusWarn)
		warnMsg := "Unable to save configuration file " + err.Error()
		if _, err := term.Writeln(color.YellowString(warnMsg)); err != nil {
			term.WriteError(err.Error())
			return false, nil
		}
		logger.Warn(warnMsg)
		return false, nil
	}

	spin.OK()
	return true, &cmdPackage

}
