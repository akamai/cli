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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func cmdUninstall(langManager packages.LangManager) cli.ActionFunc {
	return func(c *cli.Context) (e error) {
		c.Context = log.WithCommandContext(c.Context, c.Command.Name)
		logger := log.WithCommand(c.Context, c.Command.Name)
		start := time.Now()
		logger.Debug("UNINSTALL START")
		defer func() {
			if e == nil {
				logger.Debugf("UNINSTALL FINISH: %v", time.Since(start))
			} else {
				logger.Errorf("UNINSTALL ERROR: %v", e.Error())
			}
		}()
		for _, cmd := range c.Args().Slice() {
			if err := uninstallPackage(c.Context, langManager, cmd, logger); err != nil {
				logger.Error(err.Error())
				return cli.Exit(color.RedString(err.Error()), 1)
			}
		}

		return nil
	}
}

func uninstallPackage(ctx context.Context, langManager packages.LangManager, cmd string, logger log.Logger) error {
	term := terminal.Get(ctx)

	exec, _, err := findExec(ctx, langManager, cmd)
	if err != nil {
		return fmt.Errorf("command \"%s\" not found. Try \"%s help\"", cmd, tools.Self())
	}

	term.Spinner().Start(fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd))
	logger.Debugf("Attempting to uninstall \"%s\" command...", cmd)

	var repoDir string
	if len(exec) == 1 {
		repoDir = findPackageDir(filepath.Dir(exec[0]))
	} else if len(exec) > 1 {
		repoDir = findPackageDir(filepath.Dir(exec[len(exec)-1]))
	}

	if repoDir == "" {
		term.Spinner().Fail()
		logger.Error("unable to uninstall, was it installed using \"akamai install\"?")
		return fmt.Errorf("unable to uninstall, was it installed using " + color.CyanString("\"akamai install\"") + "?")
	}

	if err := os.RemoveAll(repoDir); err != nil {
		term.Spinner().Fail()
		logger.Errorf("unable to remove directory: %s", repoDir)
		return fmt.Errorf("unable to remove directory: %s", repoDir)
	}

	venvPath, err := tools.GetPkgVenvPath(fmt.Sprintf("cli-%s", cmd))
	if err != nil {
		return err
	}
	if _, err := os.Stat(venvPath); err == nil || !os.IsNotExist(err) {
		logger.Debugf("Attempting to remove package virtualenv directory")
		if err := os.RemoveAll(venvPath); err != nil {
			term.Spinner().Fail()
			logger.Errorf("unable to remove virtualenv directory: %s", venvPath)
			return fmt.Errorf("unable to remove virtualenv directory: %s", repoDir)
		}
	}

	term.Spinner().OK()

	return nil
}
