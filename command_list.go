package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func cmdList(c *cli.Context) {
	bold := color.New(color.FgWhite, color.Bold)

	color.Yellow("\nAvailable Commands:\n\n")
	for _, cmd := range getCommands() {
		for _, command := range cmd.Commands {
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
}
