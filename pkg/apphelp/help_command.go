package apphelp

import (
	"os"

	"github.com/urfave/cli/v2"
)

func cmdHelp(c *cli.Context) error {
	if c.Args().Present() {
		cmdName := c.Args().First()
		cmd := c.App.Command(cmdName)
		if cmd == nil {
			return cli.ShowAppHelp(c)
		}

		if subCmd := c.Args().Get(1); subCmd != "" || len(cmd.Subcommands) > 0 {
			os.Args = append([]string{os.Args[0], cmdName}, c.Args().Tail()...)
			os.Args = append(os.Args, "--help")
			return c.App.Run(os.Args)
		}

		if isBuiltinCommand(c, cmdName) {
			if shouldAddHelpFlag(cmd) {
				cmd.Flags = append(cmd.Flags, cli.HelpFlag)
			}
			return cli.ShowCommandHelp(c, cmdName)
		}

		os.Args = append([]string{os.Args[0], cmdName, "help"}, c.Args().Tail()...)
		return c.App.RunContext(c.Context, os.Args)
	}

	return cli.ShowAppHelp(c)
}

func hasHelpFlag(cmd *cli.Command) bool {
	for _, f := range cmd.Flags {
		if f.Names()[0] == "help" {
			return true
		}
	}
	return false
}

func shouldAddHelpFlag(cmd *cli.Command) bool {
	if !cmd.HideHelp && cli.HelpFlag != nil && !hasHelpFlag(cmd) {
		return true
	}
	return false
}

func isBuiltinCommand(c *cli.Context, cmdName string) bool {
	for _, cmd := range c.App.Commands {
		if cmd.Category != "" {
			continue
		}
		if cmd.Name == cmdName {
			return true
		}
		if contains(cmd.Aliases, cmdName) {
			return true
		}
	}

	return false
}

func contains(slc []string, e string) bool {
	for _, s := range slc {
		if s == e {
			return true
		}
	}
	return false
}
