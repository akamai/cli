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
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/version"
)

func (l *langManager) installPHP(ctx context.Context, dir, cmdReq string) error {
	bin, err := l.commandExecutor.LookPath("php")
	if err != nil {
		return fmt.Errorf("%w: %s. Please verify if the executable is included in your PATH", ErrRuntimeNotFound, "php")
	}

	logger := log.FromContext(ctx)

	logger.Debug(fmt.Sprintf("PHP binary found: %s", bin))

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := l.commandExecutor.ExecCommand(cmd)
		logger.Debug(fmt.Sprintf("%s -v: %s", bin, output))
		r := regexp.MustCompile("PHP (.*?) .*")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return fmt.Errorf("%w: %s:%s", ErrRuntimeNoVersionFound, "php", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == version.Greater {
			logger.Debug(fmt.Sprintf("PHP Version found: %s", matches[1]))
			return fmt.Errorf("%w: required: %s:%s, have: %s. Please upgrade your runtime", ErrRuntimeMinimumVersionRequired, "php", cmdReq, matches[1])
		}
	}

	return installPHPDepsComposer(ctx, l.commandExecutor, bin, dir)
}

func installPHPDepsComposer(ctx context.Context, cmdExecutor executor, phpBin, dir string) error {
	logger := log.FromContext(ctx)

	if ok, _ := cmdExecutor.FileExists(filepath.Join(dir, "composer.json")); !ok {
		return nil
	}
	logger.Info("composer.json found, running composer package manager")

	phar := filepath.Join(dir, "composer.phar")
	if ok, _ := cmdExecutor.FileExists(phar); ok {
		cmd := exec.Command(phpBin, phar, "install")
		cmd.Dir = dir
		_, err := cmdExecutor.ExecCommand(cmd)
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				logger.Debug(fmt.Sprintf("Unable to execute package manager (%s %s install): \n%s", phpBin, phar, exitErr.Stderr))
			}
			return fmt.Errorf("%w: %s", ErrPackageManagerExec, "composer")
		}
		return nil
	}

	bin, err := cmdExecutor.LookPath("composer")
	if err == nil {
		cmd := exec.Command(bin, "install")
		cmd.Dir = dir
		_, err = cmdExecutor.ExecCommand(cmd)
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				logger.Debug(fmt.Sprintf("Unable to execute package manager (%s install): \n%s", bin, exitErr.Stderr))
			}
			return fmt.Errorf("%w: %s", ErrPackageManagerExec, "composer")
		}
		return nil
	}

	bin, err = cmdExecutor.LookPath("composer.phar")
	if err == nil {
		cmd := exec.Command(bin, "install")
		cmd.Dir = dir
		_, err = cmdExecutor.ExecCommand(cmd)
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				logger.Debug(fmt.Sprintf("Unable to execute package manager (%s install): %s", bin, exitErr.Stderr))
			}
			return fmt.Errorf("%w: %s", ErrPackageManagerExec, "composer")
		}
		return nil
	}

	err = fmt.Errorf("%w: %s", ErrPackageManagerNotFound, "composer")
	logger.Debug(err.Error())
	return err
}
