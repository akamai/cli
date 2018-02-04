package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

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
			if runtime.GOOS != "windows" {
				os.Setenv("PYTHONUSERBASE", dir)
				cmd := exec.Command(bins.pip, "install", "--user", "-r", "requirements.txt")
				cmd.Dir = dir
				err = cmd.Run()
			} else {
				cmd := exec.Command(bins.pip, "install", "--isolated", "--prefix", dir, "-r", "requirements.txt")
				cmd.Dir = dir
				err = cmd.Run()
			}
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return true, nil
}

type pythonBins struct{
	python string
	pip string
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
