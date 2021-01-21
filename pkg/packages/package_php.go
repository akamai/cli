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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/akamai/cli/pkg/errors"
	akalog "github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/version"
)

// InstallPHP ..
func InstallPHP(dir, cmdReq string) (bool, error) {
	bin, err := exec.LookPath("php")
	if err != nil {
		return false, errors.NewExitErrorf(1, errors.ERRRUNTIMENOTFOUND, "PHP")
	}

	log.Tracef("PHP binary found: %s", bin)

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		log.Tracef("%s -v: %s", bin, output)
		r := regexp.MustCompile("PHP (.*?) .*")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return false, errors.NewExitErrorf(1, errors.ERRRUNTIMENOVERSIONFOUND, "PHP", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			log.Tracef("PHP Version found: %s", matches[1])
			return false, errors.NewExitErrorf(1, errors.ERRRUNTIMEMINIMUMVERSIONREQUIRED, "PHP", cmdReq, matches[1])
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
				return cli.NewExitError(errors.ERRPACKAGEMANAGEREXEC, 1)
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
				return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGEREXEC, "composer")
			}
			return nil
		}

		bin, err = exec.LookPath("composer.phar")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				akalog.Multilinef(log.Debugf, "Unable to execute package manager (%s install): %s", bin, err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGEREXEC, "composer")
			}
			return nil
		}

		log.Debugf(errors.ERRPACKAGEMANAGERNOTFOUND, "composer")
		return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGERNOTFOUND, "composer")
	}

	return nil
}
