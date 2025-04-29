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

	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/version"
)

// installRuby ...
func (l *langManager) installRuby(ctx context.Context, dir, cmdReq string) error {
	logger := log.FromContext(ctx)

	bin, err := l.commandExecutor.LookPath("ruby")
	if err != nil {
		logger.Error("Ruby executable not found")
		return fmt.Errorf("%w: %s. Please verify if the executable is included in your PATH", ErrRuntimeNotFound, "ruby")
	}

	logger.Debug(fmt.Sprintf("Ruby binary found: %s", bin))

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := l.commandExecutor.ExecCommand(cmd)
		logger.Debug(fmt.Sprintf("%s -v: %s", bin, output))
		r := regexp.MustCompile("^ruby (.*?)(p.*?) (.*)")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			logger.Error(fmt.Sprintf("Unable to determine Ruby version: %s", output))
			return fmt.Errorf("%w: %s:%s", ErrRuntimeNoVersionFound, "ruby", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == version.Greater {
			logger.Debug(fmt.Sprintf("Ruby Version found: %s", matches[1]))
			return fmt.Errorf("%w: required: %s:%s, have: %s. Please upgrade your runtime", ErrRuntimeMinimumVersionRequired, "ruby", cmdReq, matches[1])
		}
	}

	return installRubyDepsBundler(ctx, l.commandExecutor, dir)
}

func installRubyDepsBundler(ctx context.Context, cmdExecutor executor, dir string) error {
	logger := log.FromContext(ctx)

	if ok, _ := cmdExecutor.FileExists(filepath.Join(dir, "Gemfile")); !ok {
		logger.Debug("Gemfile not found")
		return nil
	}

	logger.Debug("Gemfile found, running yarn package manager")
	bin, err := cmdExecutor.LookPath("bundle")
	if err == nil {
		cmd := exec.Command(bin, "install")
		cmd.Dir = dir
		_, err = cmdExecutor.ExecCommand(cmd)
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				logger.Debug(fmt.Sprintf("Unable execute package manager (bundle install): \n%s", exitErr.Stderr))
			}
			return fmt.Errorf("%w: %s", ErrPackageManagerExec, "bundler")
		}
		return nil
	}

	err = fmt.Errorf("%w: %s", ErrPackageManagerNotFound, "bundler")
	logger.Debug(err.Error())
	return err
}
