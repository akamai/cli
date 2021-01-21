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

	log "github.com/sirupsen/logrus"

	"github.com/akamai/cli/pkg/errors"
	akalog "github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/version"
)

// InstallPython ...
func InstallPython(dir, cmdReq string) (bool, error) {
	bins, err := FindPythonBins(cmdReq)
	if err != nil {
		return false, err
	}

	if cmdReq != "" && cmdReq != "*" {
		cmd := exec.Command(bins.Python, "--version")
		output, _ := cmd.CombinedOutput()
		log.Tracef("%s --version: %s", bins.Python, output)
		r := regexp.MustCompile(`Python (\d+\.\d+\.\d+).*`)
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return false, errors.NewExitErrorf(1, errors.ERRRUNTIMENOVERSIONFOUND, "Python", cmdReq)
		}

		if version.Compare(cmdReq, matches[1]) == -1 {
			log.Tracef("Python Version found: %s", matches[1])
			return false, errors.NewExitErrorf(1, errors.ERRRUNTIMENOTFOUND, "Python")
		}
	}

	if err := installPythonDepsPip(bins, dir); err != nil {
		return false, err
	}

	return true, nil
}

// PythonBins ...
type PythonBins struct {
	Python string
	Pip    string
}

// FindPythonBins ...
func FindPythonBins(ver string) (PythonBins, error) {
	var err error

	bins := PythonBins{}
	if ver != "" && ver != "*" {
		if version.Compare("3.0.0", ver) != -1 {
			bins.Python, err = exec.LookPath("python3")
			if err != nil {
				bins.Python, err = exec.LookPath("python")
				if err != nil {
					return bins, errors.NewExitErrorf(1, errors.ERRRUNTIMENOTFOUND, "Python 3")
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
						return bins, errors.NewExitErrorf(1, errors.ERRRUNTIMENOTFOUND, "Python")
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
					return bins, errors.NewExitErrorf(1, errors.ERRRUNTIMENOTFOUND, "Python")
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

	log.Tracef("Python binary found: %s", bins.Python)
	log.Tracef("Pip binary found: %s", bins.Pip)
	return bins, nil
}

func installPythonDepsPip(bins PythonBins, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		log.Info("requirements.txt found, running pip package manager")

		if bins.Pip == "" {
			log.Debugf(errors.ERRPACKAGEMANAGERNOTFOUND, "pip")
			return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGERNOTFOUND, "pip")
		}

		if err == nil {
			os.Setenv("PYTHONUSERBASE", dir)
			args := []string{bins.Pip, "install", "--user", "--ignore-installed", "-r", "requirements.txt"}
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				akalog.Multilinef(log.Debugf, "Unable execute package manager (PYTHONUSERBASE=%s %s): \n %s", dir, strings.Join(args, " "), err.(*exec.ExitError).Stderr)
				return errors.NewExitErrorf(1, errors.ERRPACKAGEMANAGEREXEC, "pip")
			}
			return nil
		}
	}

	return nil
}
