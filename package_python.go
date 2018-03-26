/*
 Copyright 2018. Akamai Technologies, Inc

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	akamai "github.com/akamai/cli-common-golang"
	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func installPython(dir string, cmdPackage commandPackage) (bool, error) {
	bins, err := findPythonBins(cmdPackage.Requirements.Python)
	if err != nil {
		return false, err
	}

	if cmdPackage.Requirements.Python != "" && cmdPackage.Requirements.Python != "*" {
		cmd := exec.Command(bins.python, "--version")
		output, _ := cmd.CombinedOutput()
		r, _ := regexp.Compile(`Python (\d+\.\d+\.\d+).*`)
		matches := r.FindStringSubmatch(string(output))
		if versionCompare(cmdPackage.Requirements.Python, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("Python %s is required to install this command.", cmdPackage.Requirements.Python), 1)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		if bins.pip == "" {
			return false, cli.NewExitError("Unable to find package manager.", 1)
		}

		if err == nil {
			os.Setenv("PYTHONUSERBASE", dir)
			cmd := exec.Command(bins.pip, "install", "--user", "--ignore-installed", "-r", "requirements.txt")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return true, nil
}

type pythonBins struct {
	python string
	pip    string
}

func findPythonBins(version string) (pythonBins, error) {
	var err error

	bins := pythonBins{}
	if version != "" && version != "*" {
		if versionCompare("3.0.0", version) != -1 {
			bins.python, err = exec.LookPath("python3")
			if err != nil {
				bins.python, err = exec.LookPath("python")
				if err != nil {
					return bins, cli.NewExitError("Unable to locate Python 3 runtime", 1)
				}
			}
		} else {
			bins.python, err = exec.LookPath("python2")
			if err != nil {
				bins.python, err = exec.LookPath("python")
				if err != nil {
					// Even though the command specified Python 2.x, try using python3 as a last resort
					bins.python, err = exec.LookPath("python3")
					if err != nil {
						return bins, cli.NewExitError("Unable to locate Python runtime", 1)
					}
				}
			}
		}
	} else {
		bins.python, err = exec.LookPath("python3")
		if err != nil {
			bins.python, err = exec.LookPath("python2")
			if err != nil {
				bins.python, err = exec.LookPath("python")
				if err != nil {
					return bins, cli.NewExitError("Unable to locate Python runtime", 1)
				}
			}
		}
	}

	if version != "" && version != "*" {
		if versionCompare("3.0.0", version) != -1 {
			bins.pip, err = exec.LookPath("pip3")
			if err != nil {
				bins.pip, err = exec.LookPath("pip")
				if err != nil {
					return bins, nil
				}
			}
		} else {
			bins.pip, err = exec.LookPath("pip2")
			if err != nil {
				bins.pip, err = exec.LookPath("pip")
				if err != nil {
					bins.pip, err = exec.LookPath("pip3")
					if err != nil {
						return bins, nil
					}
				}
			}
		}
	} else {
		bins.pip, err = exec.LookPath("pip3")
		if err != nil {
			bins.pip, err = exec.LookPath("pip2")
			if err != nil {
				bins.pip, err = exec.LookPath("pip")
				if err != nil {
					return bins, nil
				}
			}
		}
	}

	return bins, nil
}

func migratePythonPackage(cmd string, dir string) error {
	var err error
	if runtime.GOOS == "linux" {
		_, err = os.Stat(filepath.Join(dir, ".local"))
	} else if runtime.GOOS == "darwin" {
		_, err = os.Stat(filepath.Join(dir, "Library"))
	} else if runtime.GOOS == "windows" {
		_, err = os.Stat(filepath.Join(dir, "Lib"))
	}

	if err == nil {
		fmt.Fprintln(akamai.App.Writer, color.CyanString("You must reinstall this package to continue."))
		fmt.Fprint(akamai.App.Writer, "Would you like to reinstall it? (Y/n): ")
		answer := ""
		fmt.Scanln(&answer)
		if answer != "" && strings.ToLower(answer) != "y" {
			return cli.NewExitError(color.RedString("You must reinstall this package to continue"), -1)
		}

		if err := uninstallPackage(cmd); err != nil {
			return err
		}

		if err := installPackage(cmd, false); err != nil {
			return err
		}
	}

	return nil
}
