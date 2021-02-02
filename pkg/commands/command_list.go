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

	"github.com/akamai/cli/pkg/terminal"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"

	"github.com/akamai/cli/pkg/tools"
)

func cmdList(c *cli.Context) error {
	term := terminal.Get(c.Context)
	bold := color.New(color.FgWhite, color.Bold)

	commands := listInstalledCommands(c.Context, nil, nil)

	if c.IsSet("remote") {
		packageList, err := fetchPackageList(c.Context)
		if err != nil {
			return cli.Exit("Unable to fetch remote package list", 1)
		}

		foundCommands := true
		for _, cmd := range packageList.Packages {
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
		term.Writeln(color.YellowString("\nAvailable Commands:\n\n"))

		for _, remotePackage := range packageList.Packages {
			for _, command := range remotePackage.Commands {
				if _, ok := commands[command.Name]; ok {
					continue
				}
				bold.Printf("  %s", command.Name)
				term.Writeln(fmt.Sprintf(" [package: %s]", color.BlueString(remotePackage.Name)))
				term.Printf("    %s\n", command.Description)
			}
		}

		term.Printf("\nInstall using \"%s\".\n", color.BlueString("%s install [package]", tools.Self()))
	}

	return nil
}
func listInstalledCommands(ctx context.Context, added map[string]bool, removed map[string]bool) map[string]bool {
	bold := color.New(color.FgWhite, color.Bold)

	term := terminal.Get(ctx)

	commands := make(map[string]bool)
	term.Writeln(color.YellowString("\nInstalled Commands:\n"))
	for _, cmd := range getCommands() {
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
					bold.Print(alias)
					if i < len(command.Aliases)-1 {
						term.Printf(", ")
					}
				}
				term.Printf(")")
			}

			term.Writeln()
			if len(command.Description) > 0 {
				term.Printf("    %s\n", command.Description)
			}
		}
	}
	term.Printf("\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", tools.Self()))
	return commands
}
