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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"

	"github.com/akamai/cli/pkg/errors"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/tools"
	"github.com/akamai/cli/pkg/version"
)

// installGolang ...
func installGolang(ctx context.Context, dir, cmdReq string, commands []string) error {
	logger := log.FromContext(ctx)
	bin, err := exec.LookPath("go")
	if err != nil {
		return errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "go")
	}

	logger.Debugf("Go binary found: %s", bin)

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bin, "version")
		output, _ := cmd.Output()
		logger.Debugf("%s version: %s", bin, output)
		r := regexp.MustCompile("go version go(.*?) .*")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return errors.NewExitErrorf(1, errors.ErrRuntimeNoVersionFound, "Go", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			logger.Debugf("Go Version found: %s", matches[1])
			return errors.NewExitErrorf(1, errors.ErrRuntimeMinimumVersionRequired, "Go", cmdReq, matches[1])
		}
	}

	cliPath, err := tools.GetAkamaiCliPath()
	if goPath := os.Getenv("GOPATH"); goPath != "" {
		cliPath = fmt.Sprintf("%s%d%s", goPath, os.PathListSeparator, cliPath)
	}
	if err != nil {
		return cli.Exit(color.RedString("Unable to determine CLI home directory"), 1)
	}
	if err := os.Setenv("GOPATH", cliPath); err != nil {
		return err
	}
	if err = installGolangModules(logger, dir); err != nil {
		logger.Info("go.sum not found, running glide package manager[WARN: Usage of Glide is DEPRECTED]")

		if err = installGolangDepsGlide(logger, dir); err != nil {
			return err
		}
	}

	for _, command := range commands {
		execName := "akamai-" + strings.ToLower(command)

		var cmd *exec.Cmd
		if len(commands) > 1 {
			cmd = exec.Command(bin, "build", "-o", execName, "./"+command)
		} else {
			cmd = exec.Command(bin, "build", "-o", execName, ".")
		}

		cmd.Dir = dir
		_, err = cmd.Output()
		if err != nil {
			logger.Debugf("Unable to build binary (%s): \n%s", execName, err.(*exec.ExitError).Stderr)
			return errors.NewExitErrorf(1, errors.ErrPackageCompileFailure, command)
		}
	}

	return nil
}

func installGolangDepsGlide(logger log.Logger, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "glide.lock")); err == nil {
		logger.Info("glide.lock found, running glide package manager")
		bin, err := exec.LookPath("glide")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logger.Debugf("Unable execute package manager (glide install): \n %s", err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ErrPackageManagerExec, "glide")
			}
		} else {
			logger.Debugf(errors.ErrPackageManagerNotFound, "glide")
			return errors.NewExitErrorf(1, errors.ErrPackageManagerNotFound, "glide")
		}
	}

	return nil
}

func installGolangModules(logger log.Logger, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "go.sum")); err != nil {
		return fmt.Errorf("go.sum not found")
	}
	logger.Info("go.sum found, running go module package manager")
	bin, err := exec.LookPath("go")
	if err != nil {
		logger.Debugf(errors.ErrRuntimeNotFound, "go")
		return errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "go")
	}
	cmd := exec.Command(bin, "mod", "tidy")
	cmd.Dir = dir
	_, err = cmd.Output()
	if err != nil {
		logger.Debugf("Unable execute 'go mod tidy': \n %s", err.(*exec.ExitError).Stderr)
		return errors.NewExitErrorf(1, errors.ErrPackageManagerExec, "go mod")
	}
	return nil
}
