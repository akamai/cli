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
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func cmdSubcommand(c *cli.Context) error {
	commandName := strings.ToLower(c.Command.Name)

	executable, err := findExec(commandName)
	if err != nil {
		return cli.NewExitError(color.RedString("Executable \"%s\" not found.", commandName), 1)
	}

	var packageDir string
	if len(executable) == 1 {
		packageDir = findPackageDir(executable[0])
	} else if len(executable) > 1 {
		packageDir = findPackageDir(executable[1])
	}

	cmdPackage, _ := readPackage(packageDir)

	if cmdPackage.Requirements.Python != "" {
		if err = migratePythonPackage(commandName, packageDir); err != nil {
			return err
		}

		os.Setenv("PYTHONUSERBASE", packageDir)
		if err != nil {
			return err
		}
	}

	var currentCmd command
	for _, cmd := range cmdPackage.Commands {
		if strings.ToLower(cmd.Name) == commandName {
			currentCmd = cmd
			break
		}

		for _, alias := range cmd.Aliases {
			if strings.ToLower(alias) == commandName {
				currentCmd = cmd
			}
		}
	}

	executable = append(executable, os.Args[2:]...)
	trackEvent("exec", commandName, currentCmd.Version)
	return passthruCommand(executable)
}
