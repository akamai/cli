package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func cmdList(c *cli.Context) error {
	bold := color.New(color.FgWhite, color.Bold)

	commands := make(map[string]bool)
	color.Yellow("\nInstalled Commands:\n\n")
	for _, cmd := range getCommands() {
		for _, command := range cmd.Commands {
			commands[command.Name] = true
			bold.Printf("  %s", command.Name)
			if len(command.Aliases) > 0 {
				var aliases string

				if len(command.Aliases) == 1 {
					aliases = "alias"
				} else {
					aliases = "aliases"
				}

				fmt.Printf(" (%s: ", aliases)
				for i, alias := range command.Aliases {
					bold.Print(alias)
					if i < len(command.Aliases)-1 {
						fmt.Print(", ")
					}
				}
				fmt.Print(")")
			}

			fmt.Println()

			fmt.Printf("    %s\n", command.Description)
		}
	}

	fmt.Printf("\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", self()))

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
		color.Yellow("\nAvailable Commands:\n\n")
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

				fmt.Printf(" (%s: ", aliases)
				for i, alias := range command.Aliases {
					bold.Print(alias)
					if i < len(command.Aliases)-1 {
						fmt.Print(", ")
					}
				}
				fmt.Print(")")
			}

			fmt.Println()

			fmt.Printf("    %s\n", command.Description)
		}
	}

	return nil
}
