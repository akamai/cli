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
	"time"

	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func cmdList(c *cli.Context) (e error) {
	pr := newPackageReader(embeddedPackages)
	return cmdListWithPackageReader(c, pr)
}

func cmdListWithPackageReader(c *cli.Context, pr packageReader) (e error) {
	c.Context = log.WithCommandContext(c.Context, c.Command.Name)
	start := time.Now()
	logger := log.WithCommand(c.Context, c.Command.Name)
	logger.Debug("LIST START")
	defer func() {
		if e == nil {
			logger.Debugf("LIST FINISH: %v", time.Since(start))
		} else {
			logger.Errorf("LIST ERROR: %v", e.Error())
		}
	}()
	term := terminal.Get(c.Context)
	bold := color.New(color.FgWhite, color.Bold)

	commands := listInstalledCommands(c, nil, nil)

	if c.IsSet("remote") {
		packages, err := pr.readPackage()
		if err != nil {
			return cli.Exit(fmt.Sprintf("list: %s", err), 1)
		}

		foundCommands := true
		for _, cmd := range packages.Packages {
			for _, command := range cmd.Commands {
				if _, ok := commands[command.Name]; !ok {
					foundCommands = false
					continue
				}
			}
		}

		if foundCommands {
			return nil
		}
		headerMsg := "\nAvailable Commands:\n\n"
		term.Writeln(color.YellowString(headerMsg))
		logger.Debug(headerMsg)

		for _, remotePackage := range packages.Packages {
			for _, command := range remotePackage.Commands {
				if _, ok := commands[command.Name]; ok {
					continue
				}
				commandName := bold.Sprintf("  %s", command.Name)
				term.Printf(commandName)
				packageName := fmt.Sprintf(" [package: %s]", color.BlueString(remotePackage.Name))
				term.Writeln(packageName)
				commandDescription := fmt.Sprintf("    %s\n", command.Description)
				term.Printf(commandDescription)
				logger.Debug(commandName)
				logger.Debug(packageName)
				logger.Debug(commandDescription)
			}
		}

		term.Printf("\nInstall using \"%s\".\n", color.BlueString("%s install [package]", tools.Self()))
	}

	return nil
}

func listInstalledCommands(c *cli.Context, added map[string]bool, removed map[string]bool) map[string]bool {
	bold := color.New(color.FgWhite, color.Bold)

	term := terminal.Get(c.Context)

	commands := make(map[string]bool)
	installedCmds := color.YellowString("\nInstalled Commands:\n")
	term.Writeln(installedCmds)
	cmds := getCommands(c)
	for _, cmd := range cmds {
		for _, command := range cmd.Commands {
			commands[command.Name] = true
			if _, ok := added[command.Name]; ok {
				term.Printf(color.GreenString("  %s", command.Name))
			} else if _, ok := removed[command.Name]; ok {
				term.Printf(color.RedString("  %s", command.Name))
			} else {
				term.Printf(bold.Sprintf("  %s", command.Name))
			}

			if len(command.Aliases) > 0 {
				var aliases string

				if len(command.Aliases) == 1 {
					aliases = "alias"
				} else {
					aliases = "aliases"
				}

				term.Printf(" (%s: ", aliases)
				for i, alias := range command.Aliases {
					term.Printf(bold.Sprintf(alias))
					if i < len(command.Aliases)-1 {
						term.Printf(", ")
					}
				}
				term.Printf(")")
			}

			term.Writeln()
			if len(command.Description) > 0 {
				cmdDescription := fmt.Sprintf("    %s\n", command.Description)
				term.Printf(cmdDescription)
			}
		}
	}
	term.Printf("\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", tools.Self()))
	return commands
}
