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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	log "github.com/sirupsen/logrus"
)

func installJavaScript(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("node")
	if err != nil {
		bin, err = exec.LookPath("nodejs")
		if err != nil {
			return false, NewExitErrorf(1, ERR_RUNTIME_NOT_FOUND, "Node.js")
		}
	}

	log.Tracef("Node.js binary found: %s", bin)

	if cmdPackage.Requirements.Node != "" && cmdPackage.Requirements.Node != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		log.Tracef("%s -v: %s", bin, output)
		r, _ := regexp.Compile("^v(.*?)\\s*$")
		matches := r.FindStringSubmatch(string(output))

		if len(matches) == 0 {
			return false, NewExitErrorf(1, ERR_RUNTIME_NO_VERSION_FOUND, "Node.js", cmdPackage.Requirements.Node)
		}

		if versionCompare(cmdPackage.Requirements.Node, matches[1]) == -1 {
			log.Tracef("Node.js Version found: %s", matches[1])
			return false, NewExitErrorf(1, ERR_RUNTIME_MINIMUM_VERSION_REQUIRED, "Node.js", cmdPackage.Requirements.Node, matches[1])
		}
	}

	if err := installNodeDepsYarn(dir); err != nil {
		return false, err
	}

	if err := installNodeDepsNpm(dir); err != nil {
		return false, err
	}

	return true, nil
}

func installNodeDepsYarn(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "yarn.lock")); err == nil {
		log.Info("yarn.lock found, running yarn package manager")
		bin, err := exec.LookPath("yarn")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logMultilinef(log.Debugf, "Unable execute package manager (%s install): \n%s", bin, err.(*exec.ExitError).Stderr)
				return NewExitErrorf(1, ERR_PACKAGE_MANAGER_EXEC, "yarn")
			}
			return nil
		} else {
			log.Debugf(ERR_PACKAGE_MANAGER_NOT_FOUND, "yarn")
			return NewExitErrorf(1, ERR_PACKAGE_MANAGER_NOT_FOUND, "yarn")
		}
	}

	return nil
}

func installNodeDepsNpm(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		log.Info("package.json found, running npm package manager")

		bin, err := exec.LookPath("npm")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			_, err = cmd.Output()
			if err != nil {
				logMultilinef(log.Debugf, "Unable execute package manager (%s install): \n%s", bin, err.(*exec.ExitError).Stderr)
				return NewExitErrorf(1, ERR_PACKAGE_MANAGER_EXEC, "npm")
			}
			return nil
		} else {
			log.Debugf(ERR_PACKAGE_MANAGER_NOT_FOUND, "npm")
			return NewExitErrorf(1, ERR_PACKAGE_MANAGER_NOT_FOUND, "npm")
		}
	}

	return nil
}
