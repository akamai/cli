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

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func installGolang(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("go")
	if err != nil {
		return false, cli.NewExitError("Unable to locate Go runtime", 1)
	}

	if cmdPackage.Requirements.Go != "" && cmdPackage.Requirements.Go != "*" {
		cmd := exec.Command(bin, "version")
		output, _ := cmd.Output()
		r, _ := regexp.Compile("go version go(.*?) .*")
		matches := r.FindStringSubmatch(string(output))
		if versionCompare(cmdPackage.Requirements.Go, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("Go %s is required to install this command.", cmdPackage.Requirements.Go), 1)
		}
	}

	goPath, err := getAkamaiCliPath()
	if err != nil {
		return false, cli.NewExitError(color.RedString("Unable to determine CLI home directory"), 1)
	}
	os.Setenv("GOPATH", os.Getenv("GOPATH")+string(os.PathListSeparator)+goPath)

	if _, err := os.Stat(filepath.Join(dir, "glide.lock")); err == nil {
		bin, err := exec.LookPath("glide")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, cli.NewExitError("Unable to run package manager: "+err.Error(), 1)
			}
		} else {
			return false, cli.NewExitError("Unable to find package manager.", 1)
		}
	}

	execName := "akamai-" + strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(filepath.Base(dir), "akamai-"), "cli-"))

	cmd := exec.Command(bin, "build", "-o", execName, ".")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		return false, cli.NewExitError("Unable to build binary: "+err.Error(), 1)
	}

	return true, nil
}
