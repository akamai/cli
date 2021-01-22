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

package commands

import (
	"fmt"
	"github.com/akamai/cli/pkg/io"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/tools"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func cmdUninstall(c *cli.Context) error {
	logger := log.WithCommand(c.Context, c.Command.Name)
	for _, cmd := range c.Args().Slice() {
		if err := uninstallPackage(logger, cmd); err != nil {
			stats.TrackEvent("package.uninstall", "failed", cmd)
			return err
		}
		stats.TrackEvent("package.uninstall", "success", cmd)
	}

	return nil
}

func uninstallPackage(logger log.Logger, cmd string) error {
	exec, err := findExec(logger, cmd)
	if err != nil {
		return cli.Exit(color.RedString("Command \"%s\" not found. Try \"%s help\".\n", cmd, tools.Self()), 1)
	}

	s := io.StartSpinner(fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd), fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd)+"... ["+color.GreenString("OK")+"]\n")

	var repoDir string
	if len(exec) == 1 {
		repoDir = findPackageDir(filepath.Dir(exec[0]))
	} else if len(exec) > 1 {
		repoDir = findPackageDir(filepath.Dir(exec[len(exec)-1]))
	}

	if repoDir == "" {
		io.StopSpinnerFail(s)
		return cli.Exit(color.RedString("unable to uninstall, was it installed using "+color.CyanString("\"akamai install\"")+"?"), 1)
	}

	if err := os.RemoveAll(repoDir); err != nil {
		io.StopSpinnerFail(s)
		return cli.Exit(color.RedString("unable to remove directory: %s", repoDir), 1)
	}

	io.StopSpinnerOk(s)

	return nil
}
