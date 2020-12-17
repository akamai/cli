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

package main

import (
	"fmt"
	"path/filepath"
	"strings"

	akamai "github.com/akamai/cli-common-golang"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"gopkg.in/src-d/go-git.v4"
)

func cmdUpdate(c *cli.Context) error {
	if !c.Args().Present() {
		var builtinCmds = make(map[string]bool)
		for _, cmd := range getBuiltinCommands() {
			builtinCmds[strings.ToLower(cmd.Commands[0].Name)] = true
		}

		for _, cmd := range getCommands() {
			for _, command := range cmd.Commands {
				if _, ok := builtinCmds[command.Name]; !ok {
					if err := updatePackage(command.Name, c.Bool("force")); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	for _, cmd := range c.Args() {
		if err := updatePackage(cmd, c.Bool("force")); err != nil {
			return err
		}
	}

	return nil
}

func updatePackage(cmd string, forceBinary bool) error {
	exec, err := findExec(cmd)
	if err != nil {
		return cli.NewExitError(color.RedString("Command \"%s\" not found. Try \"%s help\".\n", cmd, self()), 1)
	}

	log.Tracef("Command found: %s", filepath.Join(exec...))

	akamai.StartSpinner(fmt.Sprintf("Attempting to update \"%s\" command...", cmd), fmt.Sprintf("Attempting to update \"%s\" command...", cmd)+"... ["+color.CyanString("OK")+"]\n")

	var repoDir string
	log.Trace("Searching for package repo")
	if len(exec) == 1 {
		repoDir = findPackageDir(filepath.Dir(exec[0]))
	} else if len(exec) > 1 {
		repoDir = findPackageDir(filepath.Dir(exec[len(exec)-1]))
	}

	if repoDir == "" {
		akamai.StopSpinnerFail()
		return cli.NewExitError(color.RedString("unable to update, was it installed using "+color.CyanString("\"akamai install\"")+"?"), 1)
	}

	log.Tracef("Repo found: %s", repoDir)

	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		log.Trace("Unable to open repo")
		return cli.NewExitError(color.RedString("unable to update, there an issue with the package repo: %s", err.Error()), 1)
	}

	w, err := repo.Worktree()
	refName := "refs/remotes/" + git.DefaultRemoteName + "/master"

	refBeforePull, errBeforePull := repo.Head()
	log.Tracef("Fetching from remote: %s", git.DefaultRemoteName)
	log.Tracef("Using ref: %s", refName)

	if errBeforePull != nil {
		log.Tracef("Fetch error: %s", errBeforePull.Error())
		akamai.StopSpinnerFail()
		return cli.NewExitError(color.RedString("Unable to fetch updates (%s)", errBeforePull.Error()), 1)
	}

	err = w.Pull(&git.PullOptions{RemoteName: git.DefaultRemoteName})
	if err != nil && err.Error() != "already up-to-date" && err.Error() != "object not found" {
		log.Tracef("Fetch error: %s", err.Error())
		akamai.StopSpinnerFail()
		return cli.NewExitError(color.RedString("Unable to fetch updates (%s)", err.Error()), 1)
	}

	ref, err := repo.Head()

	if refBeforePull.Hash() != ref.Hash() {
		commit, err := repo.CommitObject(ref.Hash())
		log.Tracef("HEAD differs: %s (old) vs %s (new)", refBeforePull.Hash().String(), ref.Hash().String())
		log.Tracef("Latest commit: %s", commit)

		if err != nil && err.Error() != "already up-to-date" && err.Error() != "object not found" {
			log.Tracef("Fetch error: %s", err.Error())
			akamai.StopSpinnerFail()
			return cli.NewExitError(color.RedString("Unable to fetch updates (%s)", err.Error()), 1)
		}

	} else {
		log.Tracef("HEAD is the same as the remote: %s (old) vs %s (new)", refBeforePull.Hash().String(), ref.Hash().String())
		akamai.StopSpinnerWarnOk()
		fmt.Fprintln(akamai.App.Writer, color.CyanString("command \"%s\" already up-to-date", cmd))
		return nil
	}

	log.Tracef("Repo updated successfully")
	akamai.StopSpinnerOk()

	if !installPackageDependencies(repoDir, forceBinary) {
		log.Trace("Error updating dependencies")
		return cli.NewExitError("Unable to update command", 1)
	}

	return nil
}
