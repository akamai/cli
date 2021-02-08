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
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/akamai/cli/pkg/errors"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/version"
)

// installJavaScript ...
func installJavaScript(ctx context.Context, dir, cmdReq string) error {
	logger := log.FromContext(ctx)

	bin, err := exec.LookPath("node")
	if err != nil {
		bin, err = exec.LookPath("nodejs")
		if err != nil {
			return errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "Node.js")
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
			return errors.NewExitErrorf(1, errors.ErrRuntimeNoVersionFound, "Node.js", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			logger.Debugf("Node.js Version found: %s", matches[1])
			return errors.NewExitErrorf(1, errors.ErrRuntimeMinimumVersionRequired, "Node.js", cmdReq, matches[1])
		}
	}

	if err := installNodeDepsYarn(ctx, dir); err != nil {
		return err
	}

	if err := installNodeDepsNpm(ctx, dir); err != nil {
		return err
	}

	return nil
}

func installNodeDepsYarn(ctx context.Context, dir string) error {
	logger := log.FromContext(ctx)

	if _, err := os.Stat(filepath.Join(dir, "yarn.lock")); err == nil {
		logger.Info("yarn.lock found, running yarn package manager")
		bin, err := exec.LookPath("yarn")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logger.Debugf("Unable execute package manager (%s install): \n%s", bin, err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ErrPackageManagerExec, "yarn")
			}
			return nil
		}
		logger.Debugf(errors.ErrPackageManagerNotFound, "yarn")
		return errors.NewExitErrorf(1, errors.ErrPackageManagerNotFound, "yarn")
	}

	return nil
}

func installNodeDepsNpm(ctx context.Context, dir string) error {
	logger := log.FromContext(ctx)

	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		logger.Info("package.json found, running npm package manager")

		bin, err := exec.LookPath("npm")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logger.Debugf("Unable execute package manager (%s install): \n%s", bin, err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ErrPackageManagerExec, "npm")
			}
			return nil
		}

		logger.Debugf(errors.ErrPackageManagerNotFound, "npm")
		return errors.NewExitErrorf(1, errors.ErrPackageManagerNotFound, "npm")
	}

	return nil
}
