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

package packages

import (
	"github.com/akamai/cli/pkg/errors"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/version"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/urfave/cli/v2"
)

// InstallPHP ..
func InstallPHP(logger log.Logger, dir, cmdReq string) error {
	bin, err := exec.LookPath("php")
	if err != nil {
		return errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "PHP")
	}

	logger.Debugf("PHP binary found: %s", bin)

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		logger.Debugf("%s -v: %s", bin, output)
		r := regexp.MustCompile("PHP (.*?) .*")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return errors.NewExitErrorf(1, errors.ErrRuntimeNoVersionFound, "PHP", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			logger.Debugf("PHP Version found: %s", matches[1])
			return errors.NewExitErrorf(1, errors.ErrRuntimeMinimumVersionRequired, "PHP", cmdReq, matches[1])
		}
	}

	if err := installPHPDepsComposer(logger, bin, dir); err != nil {
		return err
	}

	return nil
}

func installPHPDepsComposer(logger log.Logger, phpBin, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "composer.json")); err == nil {
		logger.Info("composer.json found, running composer package manager")

		phar := filepath.Join(dir, "composer.phar")
		if _, err := os.Stat(phar); err == nil {
			cmd := exec.Command(phpBin, phar, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logger.Debugf("Unable to execute package manager (%s %s install): \n%s", phpBin, phar, err.(*exec.ExitError).Stderr)
				return cli.Exit(errors.ErrPackageManagerExec, 1)
			}
			return nil
		}

		bin, err := exec.LookPath("composer")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logger.Debugf("Unable to execute package manager (%s install): \n%s", bin, err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ErrPackageManagerExec, "composer")
			}
			return nil
		}

		bin, err = exec.LookPath("composer.phar")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logger.Debugf("Unable to execute package manager (%s install): %s", bin, err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ErrPackageManagerExec, "composer")
			}
			return nil
		}

		logger.Debugf(errors.ErrPackageManagerNotFound, "composer")
		return errors.NewExitErrorf(1, errors.ErrPackageManagerNotFound, "composer")
	}

	return nil
}
