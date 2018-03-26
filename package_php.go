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

	"github.com/urfave/cli"
)

func installPHP(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("php")
	if err != nil {
		return false, cli.NewExitError("Unable to locate PHP runtime", 1)
	}

	if cmdPackage.Requirements.Php != "" && cmdPackage.Requirements.Php != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		r, _ := regexp.Compile("PHP (.*?) .*")
		matches := r.FindStringSubmatch(string(output))
		if len(matches) == 0 {
			return false, cli.NewExitError(fmt.Sprintf("PHP %s is required to install this command. Unable to determine installed version.", cmdPackage.Requirements.Php), 1)
		}

		if versionCompare(cmdPackage.Requirements.Php, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("PHP %s is required to install this command.", cmdPackage.Requirements.Php), 1)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "composer.json")); err == nil {
		if _, err := os.Stat(filepath.Join(dir, "composer.phar")); err == nil {
			cmd := exec.Command(bin, filepath.Join(dir, "composer.phar"), "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}

		bin, err := exec.LookPath("composer")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}

		bin, err = exec.LookPath("composer.phar")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}

		return false, cli.NewExitError("Unable to find package manager.", 1)
	}

	return false, nil
}
