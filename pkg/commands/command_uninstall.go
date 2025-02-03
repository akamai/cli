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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/packages"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/tools"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
)

func cmdUninstall(langManager packages.LangManager) cli.ActionFunc {
	return func(c *cli.Context) (e error) {
		c.Context = log.WithCommandContext(c.Context, c.Command.Name)
		logger := log.FromContext(c.Context)
		start := time.Now()
		logger.Debug("UNINSTALL START")
		defer func() {
			if e == nil {
				logger.Debug(fmt.Sprintf("UNINSTALL FINISH: %v", time.Since(start)))
			} else {
				logger.Error(fmt.Sprintf("UNINSTALL ERROR: %v", e.Error()))
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

func uninstallPackage(ctx context.Context, langManager packages.LangManager, cmd string, logger *slog.Logger) error {
	term := terminal.Get(ctx)

	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("no home directory detected: %s", err)
	}
	home += string(filepath.Separator)
	exec, _, err := findExec(ctx, langManager, cmd)
	if err != nil {
		if !errors.Is(err, packages.ErrNoExeFound) {
			return fmt.Errorf("command \"%s\" not found. Try \"%s help\" : %s", cmd, tools.Self(), err)
		}
		// err = ErrNoExeFound - there is a directory but without any executables
		paths := filepath.SplitList(getPackageBinPaths())
		for i, path := range paths {

			// trim home directory part of a path to exclude cases where command name could be a part of it
			path = strings.TrimPrefix(path, home)
			// if trimmed path (akamai-cli defined) contains name of command to uninstall, delete directory
			if strings.Contains(path, cmd) {
				if err = os.RemoveAll(paths[i]); err != nil {
					return fmt.Errorf("could not remove directory %s: %s", paths[i], err)
				}
				return nil
			}
		}
		return fmt.Errorf("command \"%s\" not found. Try \"%s help\"", cmd, tools.Self())
	}

	term.Spinner().Start(fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd))
	logger.Debug(fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd))

	var repoDir string
	if len(exec) == 1 {
		repoDir = findPackageDir(filepath.Dir(exec[0]))
	} else if len(exec) > 1 {
		repoDir = findPackageDir(filepath.Dir(exec[len(exec)-1]))
	}

	if repoDir == "" {
		term.Spinner().Fail()
		logger.Error("unable to uninstall, was it installed using \"akamai install\"?")
		return errors.New("unable to uninstall, was it installed using " + color.CyanString("\"akamai install\"") + "?")
	}

	if err := os.RemoveAll(repoDir); err != nil {
		term.Spinner().Fail()
		logger.Error(fmt.Sprintf("unable to remove directory: %s", repoDir))
		return fmt.Errorf("unable to remove directory: %s", repoDir)
	}

	venvPath, err := tools.GetPkgVenvPath(fmt.Sprintf("cli-%s", cmd))
	if err != nil {
		return err
	}
	if _, err := os.Stat(venvPath); err == nil || !os.IsNotExist(err) {
		logger.Debug("Attempting to remove package virtualenv directory")
		if err := os.RemoveAll(venvPath); err != nil {
			term.Spinner().Fail()
			logger.Error(fmt.Sprintf("unable to remove virtualenv directory: %s", venvPath))
			return fmt.Errorf("unable to remove virtualenv directory: %s", repoDir)
		}
	}

	term.Spinner().OK()

	return nil
}
