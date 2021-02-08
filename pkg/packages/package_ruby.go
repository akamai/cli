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

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

// installRuby ...
func installRuby(ctx context.Context, dir, cmdReq string) error {
	logger := log.FromContext(ctx)

	bin, err := exec.LookPath("ruby")
	if err != nil {
		return cli.Exit(color.RedString("Unable to locate Ruby runtime"), 1)
	}

	logger.Debugf("Ruby binary found: %s", bin)

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		logger.Debugf("%s -v: %s", bin, output)
		r := regexp.MustCompile("^ruby (.*?)(p.*?) (.*)")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return errors.NewExitErrorf(1, errors.ErrRuntimeNoVersionFound, "Ruby", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			logger.Debugf("Ruby Version found: %s", matches[1])
			return errors.NewExitErrorf(1, errors.ErrRuntimeMinimumVersionRequired, "Ruby", cmdReq, matches[1])
		}
	}

	if err := installRubyDepsBundler(ctx, dir); err != nil {
		return err
	}

	return nil
}

func installRubyDepsBundler(ctx context.Context, dir string) error {
	logger := log.FromContext(ctx)

	if _, err := os.Stat(filepath.Join(dir, "Gemfile")); err == nil {
		logger.Debugf("Gemfile found, running yarn package manager")
		bin, err := exec.LookPath("bundle")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logger.Debugf("Unable execute package manager (bundle install): \n%s", err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ErrPackageManagerExec, "bundler")
			}
			return nil
		}

		logger.Debugf(errors.ErrPackageManagerNotFound, "bundler")
		return errors.NewExitErrorf(1, errors.ErrPackageManagerNotFound, "bundler")
	}

	return nil
}
