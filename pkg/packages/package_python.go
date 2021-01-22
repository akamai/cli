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
	"github.com/akamai/cli/pkg/errors"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/version"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// InstallPython ...
func InstallPython(logger log.Logger, dir, cmdReq string) error {
	bins, err := FindPythonBins(logger, cmdReq)
	if err != nil {
		return err
	}

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bins.Python, "--version")
		output, _ := cmd.CombinedOutput()
		logger.Debugf("%s --version: %s", bins.Python, output)
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

	if err := installPythonDepsPip(logger, bins, dir); err != nil {
		return err
	}

	return nil
}

// PythonBins ...
type PythonBins struct {
	Python string
	Pip    string
}

// FindPythonBins ...
func FindPythonBins(logger log.Logger, ver string) (PythonBins, error) {
	var err error

	bins := PythonBins{}
	if ver != "" && ver != "*" {
		if version.Compare("3.0.0", ver) != -1 {
			bins.Python, err = exec.LookPath("python3")
			if err != nil {
				bins.Python, err = exec.LookPath("python")
				if err != nil {
					return bins, errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "Python 3")
				}
			}
		} else {
			bins.Python, err = exec.LookPath("python2")
			if err != nil {
				bins.Python, err = exec.LookPath("python")
				if err != nil {
					// Even though the command specified Python 2.x, try using python3 as a last resort
					bins.Python, err = exec.LookPath("python3")
					if err != nil {
						return bins, errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "Python")
					}
				}
			}
		}
	} else {
		bins.Python, err = exec.LookPath("python3")
		if err != nil {
			bins.Python, err = exec.LookPath("python2")
			if err != nil {
				bins.Python, err = exec.LookPath("python")
				if err != nil {
					return bins, errors.NewExitErrorf(1, errors.ErrRuntimeNotFound, "Python")
				}
			}
		}
	}

	if ver != "" && ver != "*" {
		if version.Compare("3.0.0", ver) != -1 {
			bins.Pip, err = exec.LookPath("pip3")
			if err != nil {
				bins.Pip, err = exec.LookPath("pip")
				if err != nil {
					return bins, nil
				}
			}
		} else {
			bins.Pip, err = exec.LookPath("pip2")
			if err != nil {
				bins.Pip, err = exec.LookPath("pip")
				if err != nil {
					bins.Pip, err = exec.LookPath("pip3")
					if err != nil {
						return bins, nil
					}
				}
			}
		}
	} else {
		bins.Pip, err = exec.LookPath("pip3")
		if err != nil {
			bins.Pip, err = exec.LookPath("pip2")
			if err != nil {
				bins.Pip, err = exec.LookPath("pip")
				if err != nil {
					return bins, nil
				}
			}
		}
	}

	logger.Debugf("Python binary found: %s", bins.Python)
	logger.Debugf("Pip binary found: %s", bins.Pip)
	return bins, nil
}

func installPythonDepsPip(logger log.Logger, bins PythonBins, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err != nil {
		return nil
	}
	logger.Info("requirements.txt found, running pip package manager")

	if bins.Pip == "" {
		logger.Debugf(errors.ErrPackageManagerNotFound, "pip")
		return errors.NewExitErrorf(1, errors.ErrPackageManagerNotFound, "pip")
	}

	if err := os.Setenv("PYTHONUSERBASE", dir); err != nil {
		return err
	}
	args := []string{bins.Pip, "install", "--user", "--ignore-installed", "-r", "requirements.txt"}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if _, err := cmd.Output(); err != nil {
		logger.Debugf("Unable execute package manager (PYTHONUSERBASE=%s %s): \n %s", dir, strings.Join(args, " "), err.(*exec.ExitError).Stderr)
		return errors.NewExitErrorf(1, errors.ErrPackageManagerExec, "pip")
	}
	return nil

}
