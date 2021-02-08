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
	"strings"

	"github.com/akamai/cli/pkg/errors"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/version"
)

// installPython ...
func installPython(ctx context.Context, dir, cmdReq string) error {
	logger := log.FromContext(ctx)

	pythonBin, err := findPythonBin(ctx, cmdReq)
	if err != nil {
		return err
	}
	pipBin, err := findPipBin(ctx, cmdReq)
	if err != nil {
		return err
	}

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(pythonBin, "--version")
		output, _ := cmd.CombinedOutput()
		logger.Debugf("%s --version: %s", pythonBin, output)
		r := regexp.MustCompile(`Python (\d+\.\d+\.\d+).*`)
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return errors.NewExitErrorf(1, errors.ErrRuntimeNoVersionFound, "Python", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			logger.Debugf("Python Version found: %s", matches[1])
			return errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "Python")
		}
	}

	if err := installPythonDepsPip(ctx, pipBin, dir); err != nil {
		return err
	}

	return nil
}

// findPythonBin ...
func findPythonBin(ctx context.Context, ver string) (string, error) {
	logger := log.FromContext(ctx)

	var err error
	var bin string

	defer func() {
		if err == nil {
			logger.Debugf("Pip binary found: %s", bin)
		}
	}()
	if ver == "" || ver == "*" {
		bin, err = lookForBins("python3", "python2", "python")
		if err != nil {
			return "", errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "Python")
		}
		return bin, nil
	}
	if version.Compare("3.0.0", ver) != -1 {
		bin, err = lookForBins("python3", "python3")
		if err != nil {
			return "", errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "Python 3")
		}
		return bin, nil
	}
	bin, err = lookForBins("python2", "python", "python3")
	if err != nil {
		return "", errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "Python")
	}
	return bin, nil
}

func findPipBin(ctx context.Context, ver string) (string, error) {
	logger := log.FromContext(ctx)

	var bin string
	var err error
	defer func() {
		if err == nil {
			logger.Debugf("Pip binary found: %s", bin)
		}
	}()
	if ver == "" || ver == "*" {
		bin, err = lookForBins("pip3", "pip2", "pip")
		if err != nil {
			return "", errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "Pip")
		}
		return bin, nil
	}
	if version.Compare("3.0.0", ver) != -1 {
		bin, err = lookForBins("pip3", "pip")
		if err != nil {
			return "", errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "Pip")
		}
		return bin, nil
	}
	bin, err = lookForBins("pip2", "pip", "pip3")
	if err != nil {
		return "", errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "Pip")
	}
	return bin, nil
}

func installPythonDepsPip(ctx context.Context, bin string, dir string) error {
	logger := log.FromContext(ctx)

	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err != nil {
		return nil
	}
	logger.Info("requirements.txt found, running pip package manager")

	if bin == "" {
		logger.Debugf(errors.ErrPackageManagerNotFound, "pip")
		return errors.NewExitErrorf(1, errors.ErrPackageManagerNotFound, "pip")
	}

	if err := os.Setenv("PYTHONUSERBASE", dir); err != nil {
		return err
	}
	args := []string{bin, "install", "--user", "--ignore-installed", "-r", "requirements.txt"}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if _, err := cmd.Output(); err != nil {
		logger.Debugf("Unable execute package manager (PYTHONUSERBASE=%s %s): \n %s", dir, strings.Join(args, " "), err.(*exec.ExitError).Stderr)
		return errors.NewExitErrorf(1, errors.ErrPackageManagerExec, "pip")
	}
	return nil
}

func lookForBins(bins ...string) (string, error) {
	var err error
	for _, binName := range bins {
		bin, err := exec.LookPath(binName)
		if err == nil {
			return bin, nil
		}
	}
	return "", err
}
