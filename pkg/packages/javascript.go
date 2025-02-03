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
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/version"
)

func (l *langManager) installJavaScript(ctx context.Context, dir, ver string) error {
	logger := log.FromContext(ctx)

	bin, err := l.commandExecutor.LookPath("node")
	if err != nil {
		bin, err = l.commandExecutor.LookPath("nodejs")
		if err != nil {
			return fmt.Errorf("%w: %s. Please verify if the executable is included in your PATH", ErrRuntimeNotFound, "Node.js")
		}
	}

	logger.Debug(fmt.Sprintf("Node.js binary found: %s", bin))

	if ver != "" && ver != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := l.commandExecutor.ExecCommand(cmd)
		logger.Debug(fmt.Sprintf("%s -v: %s", bin, bytes.ReplaceAll(output, []byte("\n"), []byte(""))))
		r := regexp.MustCompile(`^v(.*?)\s*$`)
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return fmt.Errorf("%w: %s:%s", ErrRuntimeNoVersionFound, "Node.js", ver)
		}

		if version.Compare(ver, matches[1]) == version.Greater {
			logger.Debug(fmt.Sprintf("Node.js Version found: %s", matches[1]))
			return fmt.Errorf("%w: required: %s:%s, have: %s. Please upgrade your runtime", ErrRuntimeMinimumVersionRequired, "Node.js", ver, matches[1])
		}
	}

	if err := installNodeDepsYarn(ctx, l.commandExecutor, dir); err != nil {
		return err
	}

	return installNodeDepsNpm(ctx, l.commandExecutor, dir)
}

func installNodeDepsYarn(ctx context.Context, cmdExecutor executor, dir string) error {
	logger := log.FromContext(ctx)

	if ok, _ := cmdExecutor.FileExists(filepath.Join(dir, "yarn.lock")); !ok {
		return nil
	}
	logger.Info("yarn.lock found, running yarn package manager")
	bin, err := cmdExecutor.LookPath("yarn")
	if err == nil {
		cmd := exec.Command(bin, "install")
		cmd.Dir = dir
		_, err = cmdExecutor.ExecCommand(cmd)
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				logger.Debug(fmt.Sprintf("Unable execute package manager (%s install): \n%s", bin, exitErr.Stderr))
			}
			return fmt.Errorf("%w: %s", ErrPackageManagerExec, "yarn")
		}
		return nil
	}
	err = fmt.Errorf("%w: %s", ErrPackageManagerNotFound, "yarn")
	logger.Debug(err.Error())
	return err
}

func installNodeDepsNpm(ctx context.Context, cmdExecutor executor, dir string) error {
	logger := log.FromContext(ctx)

	if ok, _ := cmdExecutor.FileExists(filepath.Join(dir, "package.json")); !ok {
		return nil
	}
	logger.Info("package.json found, running npm package manager")

	bin, err := cmdExecutor.LookPath("npm")
	if err == nil {
		cmd := exec.Command(bin, "install")
		cmd.Dir = dir
		_, err = cmdExecutor.ExecCommand(cmd)
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				logger.Debug(fmt.Sprintf("Unable execute package manager (%s install): \n%s", bin, exitErr.Stderr))
			}
			return fmt.Errorf("%w: %s", ErrPackageManagerExec, "npm")
		}
		return nil
	}

	err = fmt.Errorf("%w: %s", ErrPackageManagerNotFound, "npm")
	logger.Debug(err.Error())
	return err

}
