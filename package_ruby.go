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

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func installRuby(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("ruby")
	if err != nil {
		return false, cli.NewExitError(color.RedString("Unable to locate Ruby runtime"), 1)
	}

	if cmdPackage.Requirements.Ruby != "" && cmdPackage.Requirements.Ruby != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		r, _ := regexp.Compile("^ruby (.*?)(p.*?) (.*)")
		matches := r.FindStringSubmatch(string(output))
		if versionCompare(cmdPackage.Requirements.Ruby, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("Ruby %s is required to install this command.", cmdPackage.Requirements.Ruby), 1)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "Gemfile")); err == nil {
		bin, err := exec.LookPath("bundle")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, cli.NewExitError("Unable to run package manager: "+err.Error(), 1)
			}
			return true, nil
		}
	}

	return false, cli.NewExitError("Unable to find package manager.", 1)
}
