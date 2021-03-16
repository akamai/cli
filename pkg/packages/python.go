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

	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/version"
)

func (l *langManager) installPython(ctx context.Context, dir, cmdReq string) error {
	logger := log.FromContext(ctx)

	pythonBin, err := findPythonBin(ctx, l.commandExecutor, cmdReq)
	if err != nil {
		return err
	}
	pipBin, err := findPipBin(ctx, l.commandExecutor, cmdReq)
	if err != nil {
		return err
	}

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(pythonBin, "--version")
		output, _ := l.commandExecutor.ExecCommand(cmd, true)
		logger.Debugf("%s --version: %s", pythonBin, bytes.ReplaceAll(output, []byte("\n"), []byte("")))
		r := regexp.MustCompile(`Python (\d+\.\d+\.\d+).*`)
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return fmt.Errorf("%w: %s:%s", ErrRuntimeNoVersionFound, "python", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			logger.Debugf("Python Version found: %s", matches[1])
			return fmt.Errorf("%w: required: %s:%s, have: %s", ErrRuntimeMinimumVersionRequired, "python", cmdReq, matches[1])
		}
	}

	if err := installPythonDepsPip(ctx, l.commandExecutor, pipBin, dir); err != nil {
		return err
	}

	return nil
}

func findPythonBin(ctx context.Context, cmdExecutor executor, ver string) (string, error) {
	logger := log.FromContext(ctx)

	var err error
	var bin string

	defer func() {
		if err == nil {
			logger.Debugf("Python binary found: %s", bin)
		}
	}()
	if ver == "" || ver == "*" {
		bin, err = lookForBins(cmdExecutor, "python3", "python2", "python")
		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrRuntimeNotFound, "python")
		}
		return bin, nil
	}
	if version.Compare("3.0.0", ver) != -1 {
		bin, err = lookForBins(cmdExecutor, "python3", "python")
		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrRuntimeNotFound, "python 3")
		}
		return bin, nil
	}
	bin, err = lookForBins(cmdExecutor, "python2", "python", "python3")
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrRuntimeNotFound, "python")
	}
	return bin, nil
}

func findPipBin(ctx context.Context, cmdExecutor executor, ver string) (string, error) {
	logger := log.FromContext(ctx)

	var bin string
	var err error
	defer func() {
		if err == nil {
			logger.Debugf("Pip binary found: %s", bin)
		}
	}()
	if ver == "" || ver == "*" {
		bin, err = lookForBins(cmdExecutor, "pip3", "pip2", "pip")
		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrPackageManagerNotFound, "pip")
		}
		return bin, nil
	}
	if version.Compare("3.0.0", ver) != -1 {
		bin, err = lookForBins(cmdExecutor, "pip3", "pip")
		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrPackageManagerNotFound, "pip3")
		}
		return bin, nil
	}
	bin, err = lookForBins(cmdExecutor, "pip2", "pip", "pip3")
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrPackageManagerNotFound, "pip")
	}
	return bin, nil
}

func installPythonDepsPip(ctx context.Context, cmdExecutor executor, bin, dir string) error {
	logger := log.FromContext(ctx)

	if ok, _ := cmdExecutor.FileExists(filepath.Join(dir, "requirements.txt")); !ok {
		return nil
	}
	logger.Info("requirements.txt found, running pip package manager")

	if err := os.Setenv("PYTHONUSERBASE", dir); err != nil {
		return err
	}
	args := []string{bin, "install", "--user", "--ignore-installed", "-r", "requirements.txt"}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if _, err := cmdExecutor.ExecCommand(cmd); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			logger.Debugf("Unable execute package manager (PYTHONUSERBASE=%s %s): \n %s", dir, strings.Join(args, " "), exitErr.Stderr)
		}
		return fmt.Errorf("%w: %s", ErrPackageManagerExec, "pip")
	}
	return nil
}

func lookForBins(cmdExecutor executor, bins ...string) (string, error) {
	var err error
	var bin string
	for _, binName := range bins {
		bin, err = cmdExecutor.LookPath(binName)
		if err == nil {
			return bin, nil
		}
	}
	return bin, err
}
