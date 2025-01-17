// Package autocomplete provides functions for shell autocomplete
package autocomplete

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

// Default creates the default autocomplete
func Default(ctx *cli.Context) {
	if ctx.Command.Name == "help" {
		args := []string{"akamai"}
		args = append(args, ctx.Args().Slice()...)
		args = append(args, "--"+cli.BashCompletionFlag.Names()[0])

		if err := ctx.App.RunContext(ctx.Context, args); err != nil {
			_, _ = fmt.Fprintln(ctx.App.ErrWriter, err.Error())
		}
		return
	}

	commands := make([]*cli.Command, 0)
	flags := make([]cli.Flag, 0)

	if ctx.Command.Name == "" {
		commands = ctx.App.Commands
		flags = ctx.App.VisibleFlags()
	} else {
		if len(ctx.Command.Subcommands) != 0 {
			commands = ctx.Command.Subcommands
		}

		if len(ctx.Command.VisibleFlags()) != 0 {
			flags = ctx.Command.VisibleFlags()
		}
	}

	for _, command := range commands {
		if command.Hidden {
			continue
		}

		_, _ = fmt.Fprintln(ctx.App.Writer, command.Name)
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
				_, _ = fmt.Fprintln(ctx.App.Writer, "-"+name)
			default:
				_, _ = fmt.Fprintln(ctx.App.Writer, "--"+name)
			}
		}
	}
}
