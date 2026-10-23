package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/git"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/packages"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/urfave/cli/v3"
)

func cmdSubcommand(git git.Repository, langManager packages.LangManager) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) (e error) {
		ctx = log.WithCommandContext(ctx, cmd.Name)
		logger := log.FromContext(ctx)
		term := terminal.Get(ctx)

		defer func() {
			if e != nil {
				logger.Error(fmt.Sprintf("Command execution failed: %v", e))
			} else {
				logger.Info("Command execution completed")
			}
		}()

		logger.Info(fmt.Sprintf("Executing subcommand: %s", cmd.Name))

		commandName := strings.ToLower(cmd.Name)

		executable, _, err := findExec(ctx, langManager, commandName)
		if err != nil {
			errMsg := color.RedString("Executable \"%s\" not found.", commandName)
			logger.Error(errMsg)
			return cli.Exit(errMsg, 1)
		}

		var packageDir string
		if len(executable) == 1 {
			packageDir = findPackageDir(executable[0])
		} else if len(executable) > 1 {
			packageDir = findPackageDir(executable[1])
		}

		cmdPackage, err := readInstalledPackage(packageDir)
		if err != nil {
			logger.Error(fmt.Sprintf("Error reading package: %v", err))
			return err
		}

		if cmdPackage.Requirements.Python != "" {
			exec, err := langManager.FindExec(ctx, cmdPackage.Requirements, packageDir)
			if err != nil {
				logger.Error(fmt.Sprintf("Error finding executable: %v", err))
				return err
			}

			if len(executable) == 1 {
				executable = append([]string{exec[0]}, executable...)
			} else {
				if strings.Contains(strings.ToLower(executable[0]), "python") ||
					strings.Contains(strings.ToLower(executable[0]), "py.exe") {
					executable[0] = exec[0]
				}
			}

			switch runtime.GOOS {
			case "linux":
				_, err = os.Stat(filepath.Join(packageDir, ".local"))
			case "darwin":
				_, err = os.Stat(filepath.Join(packageDir, "Library"))
			case "windows":
				_, err = os.Stat(filepath.Join(packageDir, "Lib"))
			}

			if err == nil {
				answer, err := term.Confirm("Would you like to reinstall it", true)
				logger.Debug(fmt.Sprintf("Would you like to reinstall it? %v", answer))
				if err != nil {
					logger.Error(fmt.Sprintf("Error confirming reinstall: %v", err))
					return err
				}
				if !answer {
					logger.Error(packages.ErrPackageNeedsReinstall.Error())
					return cli.Exit(color.RedString("%s", packages.ErrPackageNeedsReinstall.Error()), -1)
				}

				if err = uninstallPackage(ctx, langManager, commandName, logger); err != nil {
					return err
				}

				if _, err = installPackage(ctx, git, langManager, commandName); err != nil {
					return err
				}
			}
			if err := os.Setenv("PYTHONUSERBASE", packageDir); err != nil {
				logger.Error(fmt.Sprintf("Error setting PYTHONUSERBASE: %v", err))
				return err
			}
		}

		var currentCmd command
		for _, c := range cmdPackage.Commands {
			if strings.EqualFold(c.Name, commandName) {
				currentCmd = c
				break
			}

			for _, alias := range c.Aliases {
				if strings.EqualFold(alias, commandName) {
					currentCmd = c
				}
			}
		}

		if err := os.Setenv("AKAMAI_CLI_COMMAND", commandName); err != nil {
			logger.Error(fmt.Sprintf("Error setting AKAMAI_CLI_COMMAND: %v", err))
			return err
		}
		if err := os.Setenv("AKAMAI_CLI_COMMAND_VERSION", currentCmd.Version); err != nil {
			logger.Error(fmt.Sprintf("Error setting AKAMAI_CLI_COMMAND_VERSION: %v", err))
			return err
		}

		cmdPackage, err = readInstalledPackage(packageDir)
		if err != nil {
			logger.Error(fmt.Sprintf("Error reading package: %v", err))
			return err
		}

		if len(cmdPackage.Commands) == 0 {
			err := fmt.Errorf("package in %q does not contain any valid commands", packageDir)
			logger.Error(err.Error())
			return err
		}

		executable = prepareCommand(cmd, executable, cmd.Args().Slice(), "edgerc", "section", "accountkey")

		subCmd := createCommand(executable[0], executable[1:])
		return passthruCommand(ctx, subCmd, langManager, cmdPackage.Requirements, fmt.Sprintf("cli-%s", cmdPackage.Commands[0].Name))
	}
}

func prepareCommand(c *cli.Command, command, args []string, flags ...string) []string {
	// dont search for flags is there are no args
	if len(args) == 0 {
		return command
	}
	additionalFlags := findFlags(c, args, flags...)

	if len(command) > 1 {
		// for python or js append flags to the end
		command = append(command, args...)
		command = append(command, additionalFlags...)
	} else {
		command = append(command, additionalFlags...)
		command = append(command, args...)
	}

	return command
}

func findFlags(c *cli.Command, target []string, flags ...string) []string {
	var ret []string
	for _, flagName := range flags {
		if flagVal := c.String(flagName); flagVal != "" && !containsString(target, fmt.Sprintf("--%s", flagName)) {
			ret = append(ret, fmt.Sprintf("--%s", flagName), flagVal)
		}
	}
	return ret
}

func containsString(s []string, item string) bool {
	for _, v := range s {
		if v == item {
			return true
		}
	}
	return false
}
