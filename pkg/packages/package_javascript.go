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
)

// InstallJavaScript ...
func InstallJavaScript(logger log.Logger, dir, cmdReq string) (bool, error) {
	bin, err := exec.LookPath("node")
	if err != nil {
		bin, err = exec.LookPath("nodejs")
		if err != nil {
			return false, errors.NewExitErrorf(1, errors.ERRRUNTIMENOTFOUND, "Node.js")
		}
	}

	logger.Debugf("Node.js binary found: %s", bin)

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		logger.Debugf("%s -v: %s", bin, output)
		r := regexp.MustCompile("^v(.*?)\\s*$")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return false, errors.NewExitErrorf(1, errors.ERRRUNTIMENOVERSIONFOUND, "Node.js", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			logger.Debugf("Node.js Version found: %s", matches[1])
			return false, errors.NewExitErrorf(1, errors.ERRRUNTIMEMINIMUMVERSIONREQUIRED, "Node.js", cmdReq, matches[1])
		}
	}

	if err := installNodeDepsYarn(logger, dir); err != nil {
		return false, err
	}

	if err := installNodeDepsNpm(logger, dir); err != nil {
		return false, err
	}

	return true, nil
}

func installNodeDepsYarn(logger log.Logger, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "yarn.lock")); err == nil {
		logger.Info("yarn.lock found, running yarn package manager")
		bin, err := exec.LookPath("yarn")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logger.Debugf("Unable execute package manager (%s install): \n%s", bin, err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGEREXEC, "yarn")
			}
			return nil
		}
		logger.Debugf(errors.ERRPACKAGEMANAGERNOTFOUND, "yarn")
		return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGERNOTFOUND, "yarn")
	}

	return nil
}

func installNodeDepsNpm(logger log.Logger, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		logger.Info("package.json found, running npm package manager")

		bin, err := exec.LookPath("npm")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logger.Debugf("Unable execute package manager (%s install): \n%s", bin, err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGEREXEC, "npm")
			}
			return nil
		}

		logger.Debugf(errors.ERRPACKAGEMANAGERNOTFOUND, "npm")
		return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGERNOTFOUND, "npm")
	}

	return nil
}
