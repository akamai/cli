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

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/akamai/cli/pkg/errors"
	akalog "github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/version"
)

// InstallRuby ...
func InstallRuby(dir, cmdReq string) (bool, error) {
	bin, err := exec.LookPath("ruby")
	if err != nil {
		return false, cli.NewExitError(color.RedString("Unable to locate Ruby runtime"), 1)
	}

	log.Tracef("Ruby binary found: %s", bin)

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		log.Tracef("%s -v: %s", bin, output)
		r := regexp.MustCompile("^ruby (.*?)(p.*?) (.*)")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return false, errors.NewExitErrorf(1, errors.ERRRUNTIMENOVERSIONFOUND, "Ruby", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			log.Tracef("Ruby Version found: %s", matches[1])
			return false, errors.NewExitErrorf(1, errors.ERRRUNTIMEMINIMUMVERSIONREQUIRED, "Ruby", cmdReq, matches[1])
		}
	}

	if err := installRubyDepsBundler(dir); err != nil {
		return false, err
	}

	return true, nil
}

func installRubyDepsBundler(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "Gemfile")); err == nil {
		log.Info("Gemfile found, running yarn package manager")
		bin, err := exec.LookPath("bundle")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				akalog.Multilinef(log.Debugf, "Unable execute package manager (bundle install): \n%s", err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGEREXEC, "bundler")
			}
			return nil
		}

		log.Debugf(errors.ERRPACKAGEMANAGERNOTFOUND, "bundler")
		return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGERNOTFOUND, "bundler")
	}

	return nil
}
