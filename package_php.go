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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func installPHP(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("php")
	if err != nil {
		return false, NewExitErrorf(1, ERR_RUNTIME_NOT_FOUND, "PHP")
	}

	log.Tracef("PHP binary found: %s", bin)

	if cmdPackage.Requirements.Php != "" && cmdPackage.Requirements.Php != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		log.Tracef("%s -v: %s", bin, output)
		r, _ := regexp.Compile("PHP (.*?) .*")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return false, NewExitErrorf(1, ERR_RUNTIME_NO_VERSION_FOUND, "PHP", cmdPackage.Requirements.Php)
		}

		if versionCompare(cmdPackage.Requirements.Php, matches[1]) == -1 {
			log.Tracef("PHP Version found: %s", matches[1])
			return false, NewExitErrorf(1, ERR_RUNTIME_MINIMUM_VERSION_REQUIRED, "PHP", cmdPackage.Requirements.Php, matches[1])
		}
	}

	if err := installPHPDepsComposer(bin, dir); err != nil {
		return false, err
	}

	return true, nil
}

func installPHPDepsComposer(phpBin string, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "composer.json")); err == nil {
		log.Info("composer.json found, running composer package manager")

		phar := filepath.Join(dir, "composer.phar")
		if _, err := os.Stat(phar); err == nil {
			cmd := exec.Command(phpBin, phar, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				log.Debugf("Unable to execute package manager (%s %s install): \n%s", phpBin, phar, err.(*exec.ExitError).Stderr)
				return cli.NewExitError(ERR_PACKAGE_MANAGER_EXEC, 1)
			}
			return nil
		}

		bin, err := exec.LookPath("composer")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				log.Debugf("Unable to execute package manager (%s install): \n%s", bin, err.(*exec.ExitError).Stderr)
				return NewExitErrorf(1, ERR_PACKAGE_MANAGER_EXEC, "composer")
			}
			return nil
		}

		bin, err = exec.LookPath("composer.phar")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logMultilinef(log.Debugf, "Unable to execute package manager (%s install): %s", bin, err.(*exec.ExitError).Stderr)
				return NewExitErrorf(1, ERR_PACKAGE_MANAGER_EXEC, "composer")
			}
			return nil
		}

		log.Debugf(ERR_PACKAGE_MANAGER_NOT_FOUND, "composer")
		return NewExitErrorf(1, ERR_PACKAGE_MANAGER_NOT_FOUND, "composer")
	}

	return nil
}
