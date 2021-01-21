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
	"strings"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/akamai/cli/pkg/errors"
	akalog "github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/tools"
	"github.com/akamai/cli/pkg/version"
)

// InstallGolang ...
func InstallGolang(dir, cmdReq string, commands []string) (bool, error) {
	bin, err := exec.LookPath("go")
	if err != nil {
		return false, errors.NewExitErrorf(1, errors.ERRRUNTIMENOTFOUND, "Go")
	}

	log.Tracef("Go binary found: %s", bin)

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bin, "version")
		output, _ := cmd.Output()
		log.Tracef("%s version: %s", bin, output)
		r := regexp.MustCompile("go version go(.*?) .*")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return false, errors.NewExitErrorf(1, errors.ERRRUNTIMENOVERSIONFOUND, "Go", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			log.Tracef("Go Version found: %s", matches[1])
			return false, errors.NewExitErrorf(1, errors.ERRRUNTIMEMINIMUMVERSIONREQUIRED, "Go", cmdReq, matches[1])
		}
	}

	goPath, err := tools.GetAkamaiCliPath()
	if err != nil {
		return false, cli.NewExitError(color.RedString("Unable to determine CLI home directory"), 1)
	}
	os.Setenv("GOPATH", os.Getenv("GOPATH")+string(os.PathListSeparator)+goPath)

	// installGolangModules ...
	if err = installGolangModules(dir); err != nil {
		log.Info("go.sum not found, running glide package manager[WARN: Usage of Glide is DEPRECTED]")

		if err = installGolangDepsGlide(dir); err != nil {
			return false, err
		}
	}

	for _, command := range commands {
		execName := "akamai-" + strings.ToLower(command)

		var cmd *exec.Cmd
		if len(cmdReq) > 1 {
			cmd = exec.Command(bin, "build", "-o", execName, "./"+command)
		} else {
			cmd = exec.Command(bin, "build", "-o", execName, ".")
		}

		cmd.Dir = dir
		_, err = cmd.Output()
		if err != nil {
			akalog.Multilinef(log.Debugf, "Unable to build binary (%s): \n%s", execName, err.(*exec.ExitError).Stderr)
			return false, errors.NewExitErrorf(1, errors.ERRPACKAGECOMPILEFAILURE, command)
		}
	}

	return true, nil
}

func installGolangDepsGlide(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "glide.lock")); err == nil {
		log.Info("glide.lock found, running glide package manager")
		bin, err := exec.LookPath("glide")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				akalog.Multilinef(log.Debugf, "Unable execute package manager (glide install): \n %s", err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGEREXEC, "glide")
			}
		} else {
			log.Debugf(errors.ERRPACKAGEMANAGERNOTFOUND, "glide")
			return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGERNOTFOUND, "glide")
		}
	}

	return nil
}

func installGolangModules(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "go.sum")); err == nil {
		log.Info("go.sum found, running go module package manager")
		bin, err := exec.LookPath("go mod")
		if err == nil {
			cmd := exec.Command(bin, "tidy")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				akalog.Multilinef(log.Debugf, "Unable execute package manager (dep ensure): \n %s", err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGEREXEC, "dep")
			}
		} else {
			log.Debugf(errors.ERRPACKAGEMANAGERNOTFOUND, "dep")
			return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGERNOTFOUND, "dep")
		}
	}

	return nil
}
