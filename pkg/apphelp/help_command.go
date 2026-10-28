package apphelp

import (
	"context"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

func cmdHelp(ctx context.Context, cmd *cli.Command) error {
	root := cmd.Root()
	args := cmd.Args()
	if !args.Present() {
		return cli.ShowRootCommandHelp(root)
	}

	cmdName := args.First()
	target := root.Command(cmdName)
	if target == nil {
		return cli.ShowRootCommandHelp(root)
	}

	if isBuiltinCommand(root, cmdName) {
		if subCmdName := args.Get(1); subCmdName != "" && !strings.HasPrefix(subCmdName, "-") {
			if subCmd := target.Command(subCmdName); subCmd != nil && shouldAddHelpFlag(subCmd) {
				subCmd.Flags = append(subCmd.Flags, cli.HelpFlag)
			}
			return cli.ShowCommandHelp(ctx, target, subCmdName)
		}

		if hasVisibleSubcommands(target) {
			return cli.ShowSubcommandHelp(target)
		}
		if shouldAddHelpFlag(target) {
			target.Flags = append(target.Flags, cli.HelpFlag)
		}
		return cli.ShowCommandHelp(ctx, root, cmdName)
	}

	commandArgs := []string{os.Args[0]}
	if args.Get(1) != "" || len(target.Commands) > 0 {
		commandArgs = append(commandArgs, args.Tail()...)
		commandArgs = append(commandArgs, "--help")
	} else {
		commandArgs = append(commandArgs, "help")
		commandArgs = append(commandArgs, args.Tail()...)
	}

	command := *target
	return command.Run(ctx, commandArgs)
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

func hasVisibleSubcommands(cmd *cli.Command) bool {
	for _, subcommand := range cmd.VisibleCommands() {
		if subcommand.Name == "help" && subcommand.HideHelp && contains(subcommand.Aliases, "h") {
			continue
		}
		return true
	}
	return false
}

func isBuiltinCommand(root *cli.Command, cmdName string) bool {
	for _, cmd := range root.Commands {
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
