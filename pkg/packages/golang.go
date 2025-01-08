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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/akamai/cli/pkg/color"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/tools"
	"github.com/akamai/cli/pkg/version"
	"github.com/urfave/cli/v2"
)

func (l *langManager) installGolang(ctx context.Context, dir, ver string, commands, ldFlags []string) error {
	logger := log.FromContext(ctx)
	goBin, err := l.commandExecutor.LookPath("go")
	if err != nil {
		return fmt.Errorf("%w: %s. Please verify if the executable is included in your PATH", ErrRuntimeNotFound, "go")
	}

	logger.Debugf("Go binary found: %s", goBin)

	if ver != "" && ver != "*" {
		cmd := exec.Command(goBin, "version")
		output, _ := l.commandExecutor.ExecCommand(cmd)
		logger.Debugf("%s version: %s", goBin, bytes.ReplaceAll(output, []byte("\n"), []byte("")))
		r := regexp.MustCompile("go version go(.*?) .*")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return fmt.Errorf("%w: %s:%s", ErrRuntimeNoVersionFound, "go", ver)
		}

		if version.Compare(ver, matches[1]) == version.Greater {
			logger.Debugf("Go Version found: %s", matches[1])
			return fmt.Errorf("%w: required: %s:%s, have: %s. Please upgrade your runtime", ErrRuntimeMinimumVersionRequired, "go", ver, matches[1])
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
	if err = installGolangModules(logger, l.commandExecutor, dir); err != nil {
		logger.Info("go.sum not found, running glide package manager[WARN: Usage of Glide is DEPRECTED]")

		if err = installGolangDepsGlide(logger, l.commandExecutor, dir); err != nil {
			return err
		}
	}

	if len(commands) != len(ldFlags) {
		return fmt.Errorf("commands and ldFlags should have the same length")
	}

	for n, command := range commands {
		ldFlag := ldFlags[n]
		execName := "akamai-" + strings.ToLower(command)

		var cmd *exec.Cmd
		params := []string{"build", "-o", execName}
		if ldFlag != "" {
			params = append(params, fmt.Sprintf(`-ldflags=%s`, ldFlag))
		}
		if len(commands) > 1 {
			params = append(params, "./"+command)
		} else {
			params = append(params, ".")
		}
		cmd = exec.Command(goBin, params...)

		cmd.Dir = dir
		logger.Debugf("building with command: %+v", cmd)
		_, err = l.commandExecutor.ExecCommand(cmd)
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				logger.Debugf("Unable to build binary (%s): \n%s", execName, exitErr.Stderr)
			}
			return fmt.Errorf("%w: %s", ErrPackageCompileFailure, command)
		}
	}

	return nil
}

func installGolangDepsGlide(logger log.Logger, cmdExecutor executor, dir string) error {
	if ok, _ := cmdExecutor.FileExists(filepath.Join(dir, "glide.lock")); !ok {
		return nil
	}
	logger.Info("glide.lock found, running glide package manager")
	bin, err := cmdExecutor.LookPath("glide")
	if err == nil {
		cmd := exec.Command(bin, "install")
		cmd.Dir = dir
		_, err = cmdExecutor.ExecCommand(cmd)
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				logger.Debugf("Unable execute package manager (glide install): \n %s", exitErr.Stderr)
			}
			return fmt.Errorf("%w: %s", ErrPackageManagerExec, "glide")
		}
	} else {
		err = fmt.Errorf("%w: %s", ErrPackageManagerNotFound, "glide")
		logger.Debug(err.Error())
		return err
	}
	return nil
}

func installGolangModules(logger log.Logger, cmdExecutor executor, dir string) error {
	bin, err := cmdExecutor.LookPath("go")
	if err != nil {
		err = fmt.Errorf("%w: %s. Please verify if the executable is included in your PATH", ErrRuntimeNotFound, "go")
		logger.Debug(err.Error())
		return err
	}
	if ok, _ := cmdExecutor.FileExists(filepath.Join(dir, "go.sum")); !ok {
		dep, _ := cmdExecutor.FileExists(filepath.Join(dir, "Gopkg.lock"))
		if !dep {
			return fmt.Errorf("go.sum not found, unable to initialize go modules due to lack of Gopkg.lock")
		}
		logger.Debug("go.sum not found, attempting go mod init")
		moduleName := filepath.Base(dir)
		cmd := exec.Command(bin, "mod", "init", moduleName)
		cmd.Dir = dir
		_, err = cmdExecutor.ExecCommand(cmd)
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				logger.Debugf("Unable execute 'go mod init': \n %s", exitErr.Stderr)
			}
			return fmt.Errorf("%w: %s", ErrPackageManagerExec, "go mod init")
		}
	}
	logger.Info("go.sum found, running go module package manager")
	cmd := exec.Command(bin, "mod", "tidy")
	cmd.Dir = dir
	_, err = cmdExecutor.ExecCommand(cmd)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			logger.Debugf("Unable execute 'go mod tidy': \n %s", exitErr.Stderr)
		}
		return fmt.Errorf("%w: %s", ErrPackageManagerExec, "go mod")
	}
	return nil
}
