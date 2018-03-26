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

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func cmdList(c *cli.Context) error {
	bold := color.New(color.FgWhite, color.Bold)

	commands := make(map[string]bool)
	fmt.Fprintln(app.Writer, color.YellowString("\nInstalled Commands:\n\n"))
	for _, cmd := range getCommands() {
		for _, command := range cmd.Commands {
			commands[command.Name] = true
			fmt.Fprintf(app.Writer, bold.Sprintf("  %s", command.Name))
			if len(command.Aliases) > 0 {
				var aliases string

				if len(command.Aliases) == 1 {
					aliases = "alias"
				} else {
					aliases = "aliases"
				}

				fmt.Fprintf(app.Writer, " (%s: ", aliases)
				for i, alias := range command.Aliases {
					bold.Print(alias)
					if i < len(command.Aliases)-1 {
						fmt.Fprint(app.Writer, ", ")
					}
				}
				fmt.Fprint(app.Writer, ")")
			}

			fmt.Fprintln(app.Writer)

			fmt.Fprintf(app.Writer, "    %s\n", command.Description)
		}
	}

	fmt.Fprintf(app.Writer, "\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", self()))

	packageList, err := fetchPackageList()
	if err != nil {
		return cli.NewExitError("Unable to fetch remote package list", 1)
	}

	foundCommands := true
	for _, cmd := range packageList.Packages {
		for _, command := range cmd.Commands {
			if _, ok := commands[command.Name]; ok != true {
				foundCommands = false
				continue
			}
		}
	}

	if !foundCommands {
		fmt.Fprintln(app.Writer, color.YellowString("\nAvailable Commands:\n\n"))
	} else {
		return nil
	}

	for _, cmd := range packageList.Packages {
		for _, command := range cmd.Commands {
			if _, ok := commands[command.Name]; ok == true {
				continue
			}
			bold.Printf("  %s", command.Name)
			if len(command.Aliases) > 0 {
				var aliases string

				if len(command.Aliases) == 1 {
					aliases = "alias"
				} else {
					aliases = "aliases"
				}

				fmt.Fprintf(app.Writer, " (%s: ", aliases)
				for i, alias := range command.Aliases {
					bold.Print(alias)
					if i < len(command.Aliases)-1 {
						fmt.Fprint(app.Writer, ", ")
					}
				}
				fmt.Fprint(app.Writer, ")")
			}

			fmt.Fprintln(app.Writer)

			fmt.Fprintf(app.Writer, "    %s\n", command.Description)
		}
	}

	return nil
}
