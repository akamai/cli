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

// installRuby ...
func (l *langManager) installRuby(ctx context.Context, dir, cmdReq string) error {
	logger := log.FromContext(ctx)

	bin, err := l.commandExecutor.LookPath("ruby")
	if err != nil {
		return fmt.Errorf("%w: %s", ErrRuntimeNotFound, "ruby")
	}

	logger.Debugf("Ruby binary found: %s", bin)

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := l.commandExecutor.ExecCommand(cmd)
		logger.Debugf("%s -v: %s", bin, output)
		r := regexp.MustCompile("^ruby (.*?)(p.*?) (.*)")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return fmt.Errorf("%w: %s:%s", ErrRuntimeNoVersionFound, "ruby", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			logger.Debugf("Ruby Version found: %s", matches[1])
			return fmt.Errorf("%w: required: %s:%s, have: %s", ErrRuntimeMinimumVersionRequired, "ruby", cmdReq, matches[1])
		}
	}

	if err := installRubyDepsBundler(ctx, l.commandExecutor, dir); err != nil {
		return err
	}

	return nil
}

func installRubyDepsBundler(ctx context.Context, cmdExecutor executor, dir string) error {
	logger := log.FromContext(ctx)

	if ok, _ := cmdExecutor.FileExists(filepath.Join(dir, "Gemfile")); !ok {
		return nil
	}
	logger.Debugf("Gemfile found, running yarn package manager")
	bin, err := cmdExecutor.LookPath("bundle")
	if err == nil {
		cmd := exec.Command(bin, "install")
		cmd.Dir = dir
		_, err = cmdExecutor.ExecCommand(cmd)
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				logger.Debugf("Unable execute package manager (bundle install): \n%s", exitErr.Stderr)
			}
			return fmt.Errorf("%w: %s", ErrPackageManagerExec, "bundler")
		}
		return nil
	}

	err = fmt.Errorf("%w: %s", ErrPackageManagerNotFound, "bundler")
	logger.Debug(err.Error())
	return err
}
