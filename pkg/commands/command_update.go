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
	"reflect"
	"strings"
	"time"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/git"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/packages"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/tools"
	gogit "github.com/go-git/go-git/v5"
	"github.com/urfave/cli/v2"
)

func cmdUpdate(gitRepo git.Repository, langManager packages.LangManager) cli.ActionFunc {
	return func(c *cli.Context) (e error) {
		c.Context = log.WithCommandContext(c.Context, c.Command.Name)
		logger := log.FromContext(c.Context)
		start := time.Now()
		logger.Debug("UPDATE START")
		defer func() {
			if e == nil {
				logger.Debug(fmt.Sprintf("UPDATE FINISH: %v", time.Since(start)))
			} else {
				logger.Error(fmt.Sprintf("UPDATE ERROR: %v", e))
			}
		}()
		if !c.Args().Present() {
			var builtinCmds = make(map[string]bool)
			for _, cmd := range getBuiltinCommands(c) {
				builtinCmds[strings.ToLower(cmd.Commands[0].Name)] = true
			}

			for _, cmd := range getCommands(c) {
				for _, command := range cmd.Commands {
					if _, ok := builtinCmds[command.Name]; !ok {
						if err := updatePackage(c.Context, gitRepo, langManager, logger, command.Name); err != nil {
							logger.Error(fmt.Sprintf("Error updating package: %v", err))
							return err
						}
					}
				}
			}

			return nil
		}

		for _, cmd := range c.Args().Slice() {
			if err := updatePackage(c.Context, gitRepo, langManager, logger, cmd); err != nil {
				logger.Error(fmt.Sprintf("Error updating package: %v", err))
				return err
			}
		}

		return nil
	}
}

func updatePackage(ctx context.Context, gitRepo git.Repository, langManager packages.LangManager, logger *slog.Logger, cmd string) error {
	term := terminal.Get(ctx)
	exec, _, err := findExec(ctx, langManager, cmd)
	if err != nil {
		logger.Error(fmt.Sprintf("Command \"%s\" not found: %v", cmd, err))
		return cli.Exit(color.RedString("Command \"%s\" not found. Try \"%s help\".\n", cmd, tools.Self()), 1)
	}

	logger.Debug(fmt.Sprintf("Command found: %s", filepath.Join(exec...)))

	term.Spinner().Start("Attempting to update \"%s\" command...", cmd)

	var repoDir string
	logger.Debug("Searching for package repo")
	if len(exec) == 1 {
		repoDir = findPackageDir(filepath.Dir(exec[0]))
	} else if len(exec) > 1 {
		repoDir = findPackageDir(filepath.Dir(exec[len(exec)-1]))
	}

	if repoDir == "" {
		term.Spinner().Fail()
		logger.Error("Unable to find package directory")
		return cli.Exit(color.RedString("unable to update, was it installed using %s", color.CyanString("\"akamai install\"")+"?"), 1)
	}

	logger.Debug(fmt.Sprintf("Repo found: %s", repoDir))

	err = gitRepo.Open(repoDir)
	if err != nil {
		logger.Debug("Unable to open repo")

		cmdPackage, err := readPackage(repoDir)
		if err != nil {
			term.Spinner().Fail()
			logger.Error(fmt.Sprintf("Failed to read package: %v", err))
			return cli.Exit(color.RedString("unable to update, there was an issue with the package repo: %v", err), 1)
		}

		packageVersions := map[string]string{}
		for _, command := range cmdPackage.Commands {
			packageVersions[command.Name] = command.Version
		}

		repo := filepath.Base(repoDir)
		url := fmt.Sprintf(githubRawURLTemplate, repo)

		remotePackage, err := readPackageFromGithub(url, repoDir)
		if err != nil {
			term.Spinner().Fail()
			logger.Error(fmt.Sprintf("Failed to read package from github: %v", err))
			return cli.Exit(color.RedString("unable to update, there was an issue with fetching latest configuration file: %v", err), 1)
		}

		remoteVersions := map[string]string{}
		for _, command := range remotePackage.Commands {
			remoteVersions[command.Name] = command.Version
		}

		if reflect.DeepEqual(packageVersions, remoteVersions) {
			term.Spinner().WarnOK()
			debugMessage := fmt.Sprintf("command \"%s\" already up-to-date", cmd)
			logger.Warn(debugMessage)
			if _, err := term.Writeln(color.CyanString("%s", debugMessage)); err != nil {
				term.WriteError(err.Error())
				return err
			}
			return nil
		}

		tempDir := filepath.Dir(repoDir) + "/.tmp_" + filepath.Base(repoDir)
		logger.Debug(fmt.Sprintf("Moving package to temporary dir: %s", tempDir))
		if err = os.Rename(repoDir, tempDir); err != nil {
			term.Spinner().Fail()
			logger.Error(fmt.Sprintf("Unable to move package to temporary dir: %v", err))
			return cli.Exit(color.RedString("unable to update, there was an issue with the package repo: %v", err), 1)
		}

		logger.Debug(fmt.Sprintf("Attempting to install package: %s", cmd))
		_, err = installPackage(ctx, gitRepo, langManager, tools.Githubize(cmd))
		if err != nil {
			term.Spinner().Fail()
			if err := os.Rename(tempDir, repoDir); err != nil {
				logger.Error(fmt.Sprintf("Unable to move package back to original dir: %v", err))
				return cli.Exit(color.RedString("unable to update, there was an issue with the package repo: %v", err), 1)
			}
			logger.Error(fmt.Sprintf("Failed to install package: %v", err))
			return cli.Exit(color.RedString("unable to update: %v", err), 1)
		}

		if err := os.RemoveAll(tempDir); err != nil {
			term.Spinner().Fail()
			logger.Error(fmt.Sprintf("Unable to remove temporary dir: %v", err))
			return cli.Exit(color.RedString("unable to update, there was an issue with the package repo: %v", err), 1)
		}

		term.Spinner().OK()
		logger.Debug("Repo updated successfully")

		return nil
	}

	err = updateRepo(ctx, gitRepo, logger, term, cmd)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to update repo: %v", err))
		return err
	}

	if ok, _ := installPackageDependencies(ctx, langManager, repoDir, logger); !ok {
		term.Spinner().Fail()
		logger.Debug("Error updating dependencies")
		return cli.Exit("Unable to update command", 1)
	}

	term.Spinner().OK()
	logger.Debug("Repo updated successfully")

	return nil
}

func updateRepo(ctx context.Context, gitRepo git.Repository, logger *slog.Logger, term terminal.Terminal, cmd string) error {
	w, err := gitRepo.Worktree()
	if err != nil {
		term.Spinner().Fail()
		logger.Error("Unable to open repo")
		return cli.Exit(color.RedString("unable to update, there was an issue with the package repo: %v", err), 1)
	}

	if err := gitRepo.Reset(&gogit.ResetOptions{Mode: gogit.HardReset}); err != nil {
		term.Spinner().Warn()
		logger.Error(fmt.Sprintf("Unable to reset the branch changes: %v", err))
		if _, err := term.Writeln(color.YellowString("unable to reset the branch changes, we will try to continue anyway: %v", err)); err != nil {
			return err
		}
	}

	refName := "refs/remotes/" + git.DefaultRemoteName + "/master"

	refBeforePull, errBeforePull := gitRepo.Head()
	logger.Debug(fmt.Sprintf("Fetching from remote: %s", git.DefaultRemoteName))
	logger.Debug(fmt.Sprintf("Using ref: %s", refName))

	if errBeforePull != nil {
		term.Spinner().Fail()
		logger.Error(fmt.Sprintf("Fetch error: %v", errBeforePull))
		return cli.Exit(color.RedString("Unable to fetch updates: %v", errBeforePull), 1)
	}

	err = gitRepo.Pull(ctx, w)
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		term.Spinner().Fail()
		logger.Error(fmt.Sprintf("Fetch error: %v", err))
		return cli.Exit(color.RedString("%s", tools.CapitalizeFirstWord(err.Error())), 1)
	}

	ref, err := gitRepo.Head()
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		term.Spinner().Fail()
		logger.Error(fmt.Sprintf("Fetch error: %v", err))
		return cli.Exit(color.RedString("Unable to fetch updates: %v", err), 1)
	}

	if refBeforePull.Hash() != ref.Hash() {
		commit, err := gitRepo.CommitObject(ref.Hash())
		logger.Debug(fmt.Sprintf("HEAD differs: %s (old) vs %s (new)", refBeforePull.Hash().String(), ref.Hash().String()))
		logger.Debug(fmt.Sprintf("Latest commit: %s", commit))

		if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
			term.Spinner().Fail()
			logger.Error(fmt.Sprintf("Fetch error: %v", err))
			return cli.Exit(color.RedString("Unable to fetch updates: %v", err), 1)
		}
	} else {
		logger.Debug(fmt.Sprintf("HEAD is the same as the remote: %s (old) vs %s (new)", refBeforePull.Hash().String(), ref.Hash().String()))
		debugMessage := fmt.Sprintf("command \"%s\" already up-to-date", cmd)
		logger.Warn(debugMessage)
		if _, err := term.Writeln(color.CyanString("%s", debugMessage)); err != nil {
			return err
		}
	}

	return nil
}
