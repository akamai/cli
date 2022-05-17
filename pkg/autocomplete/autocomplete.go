package autocomplete

import (
	"os"
	"strings"

	"github.com/akamai/cli/pkg/terminal"

	"github.com/urfave/cli/v2"
)

// Default creates the default autocomplete
func Default(ctx *cli.Context) {
	term := terminal.Get(ctx.Context)
	if ctx.Command.Name == "help" {
		var args []string
		args = append(args, os.Args[0])
		if len(os.Args) > 2 {
			args = append(args, os.Args[2:]...)
		}

		if err := ctx.App.Run(args); err != nil {
			term.WriteError(err.Error())
		}
	}

	commands := make([]*cli.Command, 0)
	flags := make([]cli.Flag, 0)

	if ctx.Command.Name == "" {
		commands = ctx.App.Commands
		flags = ctx.App.Flags
	} else {
		if len(ctx.Command.Subcommands) != 0 {
			commands = ctx.Command.Subcommands
		}

		if len(ctx.Command.Flags) != 0 {
			flags = ctx.Command.Flags
		}
	}

	for _, command := range commands {
		if command.Hidden {
			continue
		}

		for _, name := range command.Names() {
			term.Writeln(ctx.App.Writer, name)
		}
	}

	for _, flag := range flags {
	nextFlag:
		for _, name := range flag.Names() {
			name = strings.TrimSpace(name)

			if len(cli.BashCompletionFlag.Names()) > 0 && name == cli.BashCompletionFlag.Names()[0] {
				continue
			}

			for _, arg := range os.Args {
				if arg == "--"+name || arg == "-"+name {
					continue nextFlag
				}
			}

			switch len(name) {
			case 0:
			case 1:
				term.Writeln(ctx.App.Writer, "-"+name)
			default:
				term.Writeln(ctx.App.Writer, "--"+name)
			}
		}
	}
}
