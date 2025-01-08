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
	"reflect"
	"strings"
	"time"

	"github.com/akamai/cli/pkg/color"
	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	gogit "github.com/go-git/go-git/v5"
	"github.com/urfave/cli/v2"
)

func cmdUpdate(gitRepo git.Repository, langManager packages.LangManager) cli.ActionFunc {
	return func(c *cli.Context) (e error) {
		c.Context = log.WithCommandContext(c.Context, c.Command.Name)
		logger := log.WithCommand(c.Context, c.Command.Name)
		start := time.Now()
		logger.Debug("UPDATE START")
		defer func() {
			if e == nil {
				logger.Debugf("UPDATE FINISH: %v", time.Since(start))
			} else {
				logger.Errorf("UPDATE ERROR: %v", e.Error())
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
							return err
						}
					}
				}
			}

			return nil
		}

		for _, cmd := range c.Args().Slice() {
			if err := updatePackage(c.Context, gitRepo, langManager, logger, cmd); err != nil {
				return err
			}
		}

		return nil
	}
}

func updatePackage(ctx context.Context, gitRepo git.Repository, langManager packages.LangManager, logger log.Logger, cmd string) error {
	term := terminal.Get(ctx)
	exec, _, err := findExec(ctx, langManager, cmd)
	if err != nil {
		return cli.Exit(color.RedString("Command \"%s\" not found. Try \"%s help\".\n", cmd, tools.Self()), 1)
	}

	logger.Debugf("Command found: %s", filepath.Join(exec...))

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
		return cli.Exit(color.RedString("unable to update, was it installed using "+color.CyanString("\"akamai install\"")+"?"), 1)
	}

	logger.Debugf("Repo found: %s", repoDir)

	err = gitRepo.Open(repoDir)
	if err != nil {

		logger.Debug("Unable to open repo")

		cmdPackage, err := readPackage(repoDir)
		if err != nil {
			return cli.Exit(color.RedString("unable to update, there was an issue with the package repo: %s", err.Error()), 1)
		}

		packageVersions := map[string]string{}
		for _, command := range cmdPackage.Commands {
			packageVersions[command.Name] = command.Version
		}

		repo := filepath.Base(repoDir)
		url := fmt.Sprintf(githubRawURLTemplate, repo)

		remotePackage, err := readPackageFromGithub(url, repoDir)
		if err != nil {
			return cli.Exit(color.RedString("unable to update, there was an issue with fetching latest configuration file: %s", err.Error()), 1)
		}

		remoteVersions := map[string]string{}
		for _, command := range remotePackage.Commands {
			remoteVersions[command.Name] = command.Version
		}

		if reflect.DeepEqual(packageVersions, remoteVersions) {
			term.Spinner().WarnOK()
			debugMessage := fmt.Sprintf("command \"%s\" already up-to-date", cmd)
			logger.Warn(debugMessage)
			if _, err := term.Writeln(color.CyanString(debugMessage)); err != nil {
				return err
			}
			return nil
		}

		tempDir := filepath.Dir(repoDir) + "/.tmp_" + filepath.Base(repoDir)
		logger.Debugf("Moving package to temporary dir: %s", tempDir)
		if err = os.Rename(repoDir, tempDir); err != nil {
			return cli.Exit(color.RedString("unable to update, there was an issue with the package repo: %s", err.Error()), 1)
		}

		_, err = installPackage(ctx, gitRepo, langManager, tools.Githubize(cmd))
		if err != nil {
			term.Spinner().Fail()
			if err := os.Rename(tempDir, repoDir); err != nil {
				return cli.Exit(color.RedString("unable to update, there was an issue with the package repo: %s", err.Error()), 1)
			}
			return cli.Exit(color.RedString("unable to update: %s", err.Error()), 1)
		}

		if err := os.RemoveAll(tempDir); err != nil {
			return cli.Exit(color.RedString("unable to update, there was an issue with the package repo: %s", err.Error()), 1)
		}

		logger.Debug("Repo updated successfully")
		term.Spinner().OK()
		return nil

	}

	err = updateRepo(ctx, gitRepo, logger, term, cmd)
	if err != nil {
		return err
	}

	if ok, _ := installPackageDependencies(ctx, langManager, repoDir, logger); !ok {
		logger.Trace("Error updating dependencies")
		return cli.Exit("Unable to update command", 1)
	}
	logger.Debug("Repo updated successfully")
	term.Spinner().OK()

	return nil
}

func updateRepo(ctx context.Context, gitRepo git.Repository, logger log.Logger, term terminal.Terminal, cmd string) error {
	w, err := gitRepo.Worktree()
	if err != nil {
		logger.Debug("Unable to open repo")
		term.Spinner().Fail()
		return cli.Exit(color.RedString("unable to update, there was an issue with the package repo: %s", err.Error()), 1)
	}

	if err := gitRepo.Reset(&gogit.ResetOptions{Mode: gogit.HardReset}); err != nil {
		logger.Debug(err.Error())
		term.Spinner().Warn()
		if _, err := term.Writeln(color.YellowString("unable to reset the branch changes, we will try to continue anyway: %s", err.Error())); err != nil {
			return err
		}
	}

	refName := "refs/remotes/" + git.DefaultRemoteName + "/master"

	refBeforePull, errBeforePull := gitRepo.Head()
	logger.Debugf("Fetching from remote: %s", git.DefaultRemoteName)
	logger.Debugf("Using ref: %s", refName)

	if errBeforePull != nil {
		logger.Debugf("Fetch error: %s", errBeforePull.Error())
		term.Spinner().Fail()
		return cli.Exit(color.RedString("Unable to fetch updates (%s)", errBeforePull.Error()), 1)
	}

	err = gitRepo.Pull(ctx, w)
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		logger.Debug(tools.CapitalizeFirstWord(err.Error()))
		term.Spinner().Fail()
		return cli.Exit(color.RedString(tools.CapitalizeFirstWord(err.Error())), 1)
	}

	ref, err := gitRepo.Head()
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		logger.Debugf("Fetch error: %s", err.Error())
		term.Spinner().Fail()
		return cli.Exit(color.RedString("Unable to fetch updates (%s)", err.Error()), 1)
	}

	if refBeforePull.Hash() != ref.Hash() {
		commit, err := gitRepo.CommitObject(ref.Hash())
		logger.Debugf("HEAD differs: %s (old) vs %s (new)", refBeforePull.Hash().String(), ref.Hash().String())
		logger.Debugf("Latest commit: %s", commit)

		if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
			logger.Debugf("Fetch error: %s", err.Error())
			term.Spinner().Fail()
			return cli.Exit(color.RedString("Unable to fetch updates (%s)", err.Error()), 1)
		}
	} else {
		logger.Debugf("HEAD is the same as the remote: %s (old) vs %s (new)", refBeforePull.Hash().String(), ref.Hash().String())
		term.Spinner().WarnOK()
		debugMessage := fmt.Sprintf("command \"%s\" already up-to-date", cmd)
		logger.Warn(debugMessage)
		if _, err := term.Writeln(color.CyanString(debugMessage)); err != nil {
			return err
		}
	}
	return nil
}
