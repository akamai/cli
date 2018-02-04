package main

import (
	"os"

	"github.com/urfave/cli"
)

func cmdHelp(c *cli.Context) error {
	if c.Args().Present() {
		cmd := c.Args().First()

		builtinCmds := getBuiltinCommands()
		for _, builtInCmd := range builtinCmds {
			if builtInCmd.Commands[0].Name == cmd {
				return cli.ShowCommandHelp(c, cmd)
			}
		}

		// The arg mangling ensures that aliases are handled
		os.Args = append([]string{os.Args[0], cmd, "help"}, c.Args().Tail()...)
		main()
		return nil
	}

	return cli.ShowAppHelp(c)
}