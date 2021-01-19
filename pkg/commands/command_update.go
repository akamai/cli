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
	"fmt"
	"github.com/akamai/cli/pkg/app"
	"github.com/akamai/cli/pkg/io"
	"github.com/akamai/cli/pkg/tools"
	"path/filepath"
	"strings"

	"github.com/akamai/cli/pkg/log"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"gopkg.in/src-d/go-git.v4"
)

func cmdUpdate(c *cli.Context) error {
	logger := log.WithCommand(c.Context, c.Command.Name)
	if !c.Args().Present() {
		var builtinCmds = make(map[string]bool)
		for _, cmd := range getBuiltinCommands() {
			builtinCmds[strings.ToLower(cmd.Commands[0].Name)] = true
		}

		for _, cmd := range getCommands() {
			for _, command := range cmd.Commands {
				if _, ok := builtinCmds[command.Name]; !ok {
					if err := updatePackage(logger, command.Name, c.Bool("force")); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	for _, cmd := range c.Args().Slice() {
		if err := updatePackage(logger, cmd, c.Bool("force")); err != nil {
			return err
		}
	}

	return nil
}

func updatePackage(logger log.Logger, cmd string, forceBinary bool) error {
	exec, err := findExec(logger, cmd)
	if err != nil {
		return cli.NewExitError(color.RedString("Command \"%s\" not found. Try \"%s help\".\n", cmd, tools.Self()), 1)
	}

	logger.Debugf("Command found: %s", filepath.Join(exec...))

	s := io.StartSpinner(fmt.Sprintf("Attempting to update \"%s\" command...", cmd), fmt.Sprintf("Attempting to update \"%s\" command...", cmd)+"... ["+color.CyanString("OK")+"]\n")

	var repoDir string
	logger.Debug("Searching for package repo")
	if len(exec) == 1 {
		repoDir = findPackageDir(filepath.Dir(exec[0]))
	} else if len(exec) > 1 {
		repoDir = findPackageDir(filepath.Dir(exec[len(exec)-1]))
	}

	if repoDir == "" {
		io.StopSpinnerFail(s)
		return cli.NewExitError(color.RedString("unable to update, was it installed using "+color.CyanString("\"akamai install\"")+"?"), 1)
	}

	logger.Debugf("Repo found: %s", repoDir)

	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		logger.Debug("Unable to open repo")
		return cli.NewExitError(color.RedString("unable to update, there an issue with the package repo: %s", err.Error()), 1)
	}

	w, err := repo.Worktree()
	refName := "refs/remotes/" + git.DefaultRemoteName + "/master"

	refBeforePull, errBeforePull := repo.Head()
	logger.Debugf("Fetching from remote: %s", git.DefaultRemoteName)
	logger.Debugf("Using ref: %s", refName)

	if errBeforePull != nil {
		logger.Debugf("Fetch error: %s", errBeforePull.Error())
		io.StopSpinnerFail(s)
		return cli.NewExitError(color.RedString("Unable to fetch updates (%s)", errBeforePull.Error()), 1)
	}

	err = w.Pull(&git.PullOptions{RemoteName: git.DefaultRemoteName})
	if err != nil && err.Error() != "already up-to-date" && err.Error() != "object not found" {
		logger.Debugf("Fetch error: %s", err.Error())
		io.StopSpinnerFail(s)
		return cli.NewExitError(color.RedString("Unable to fetch updates (%s)", err.Error()), 1)
	}

	ref, err := repo.Head()

	if refBeforePull.Hash() != ref.Hash() {
		commit, err := repo.CommitObject(ref.Hash())
		logger.Debugf("HEAD differs: %s (old) vs %s (new)", refBeforePull.Hash().String(), ref.Hash().String())
		logger.Debugf("Latest commit: %s", commit)

		if err != nil && err.Error() != "already up-to-date" && err.Error() != "object not found" {
			logger.Debugf("Fetch error: %s", err.Error())
			io.StopSpinnerFail(s)
			return cli.NewExitError(color.RedString("Unable to fetch updates (%s)", err.Error()), 1)
		}

	} else {
		logger.Debugf("HEAD is the same as the remote: %s (old) vs %s (new)", refBeforePull.Hash().String(), ref.Hash().String())
		io.StopSpinnerWarnOk(s)
		fmt.Fprintln(app.App.Writer, color.CyanString("command \"%s\" already up-to-date", cmd))
		return nil
	}

	logger.Debug("Repo updated successfully")
	io.StopSpinnerOk(s)

	if !installPackageDependencies(logger, repoDir, forceBinary) {
		logger.Debug("Error updating dependencies")
		return cli.NewExitError("Unable to update command", 1)
	}

	return nil
}
