// Package autocomplete provides functions for shell autocomplete
package autocomplete

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

// Default creates the default autocomplete.
func Default(_ context.Context, cmd *cli.Command) {
	root := cmd.Root()
	target := cmd

	if cmd.Name == "help" {
		target = root
		for _, name := range cmd.Args().Slice() {
			next := target.Command(name)
			if next == nil {
				break
			}
			target = next
		}
	}

	for _, command := range target.Commands {
		if command.Hidden {
			continue
		}

		_, _ = fmt.Fprintln(root.Writer, command.Name)
	}

	helpFlagName := ""
	if cli.HelpFlag != nil {
		if names := cli.HelpFlag.Names(); len(names) > 0 {
			helpFlagName = names[0]
		}
	}
	hideHelpFlagSuggestion := target != root && target.HideHelpCommand

	for _, flag := range target.VisibleFlags() {
		flagNames := flag.Names()
		if helpFlagName != "" && len(flagNames) > 0 && flagNames[0] == helpFlagName {
			if hideHelpFlagSuggestion {
				continue
			}
			flagNames = flagNames[:1]
		}

	nextFlag:
		for _, name := range flagNames {
			name = strings.TrimSpace(name)

			if len(cli.GenerateShellCompletionFlag.Names()) > 0 && name == cli.GenerateShellCompletionFlag.Names()[0] {
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
				_, _ = fmt.Fprintln(root.Writer, "-"+name)
			default:
				_, _ = fmt.Fprintln(root.Writer, "--"+name)
			}
		}
	}
}
