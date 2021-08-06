package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/akamai/cli/pkg/app"
	"github.com/akamai/cli/pkg/commands"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/akamai/cli/pkg/version"
	"github.com/urfave/cli/v2"
)

// Run is the entry point to the CLI
func Run() int {
	ctx := context.Background()
	term := terminal.Color()
	logger := log.FromContext(ctx)

	var pathErr *os.PathError
	if err := cleanupUpgrade(); err != nil && errors.As(err, &pathErr) && pathErr.Err != syscall.ENOENT {
		logger.Debugf("Unable to remove old executable: %s", err.Error())
	}

	if err := os.Setenv("AKAMAI_CLI", "1"); err != nil {
		term.WriteErrorf("Unable to set AKAMAI_CLI: %s", err.Error())
		return 1
	}
	if err := os.Setenv("AKAMAI_CLI_VERSION", version.Version); err != nil {
		term.WriteErrorf("Unable to set AKAMAI_CLI_VERSION: %s", err.Error())
		return 1
	}
	cfg, err := config.NewIni()
	if err != nil {
		term.WriteErrorf("Unable to open cli config: %s", err.Error())
		return 2
	}
	ctx = config.Context(ctx, cfg)

	cachePath, ok := cfg.GetValue("cli", "cache-path")
	if !ok {
		cliHome, _ := tools.GetAkamaiCliPath()

		cachePath = filepath.Join(cliHome, "cache")
		if err := os.MkdirAll(cachePath, 0700); err != nil {
			term.WriteErrorf("Unable to create cache directory: %s", err.Error())
			return 2
		}
	}

	ctx = terminal.Context(ctx, term)

	cfg.SetValue("cli", "cache-path", cachePath)
	if err := cfg.Save(ctx); err != nil {
		return 3
	}
	if err := cfg.ExportEnv(ctx); err != nil {
		term.WriteErrorf("Unable to export required envs: %s", err.Error())
	}

	cliApp := app.CreateApp(ctx)
	ctx = log.SetupContext(ctx, cliApp.Writer)

	cmds := commands.CommandLocator(ctx)
	cliApp.Commands = cmds

	if err := firstRun(ctx); err != nil {
		return 5
	}
	checkUpgrade(ctx)
	if err := stats.CheckPing(ctx); err != nil {
		term.WriteError(err.Error())
	}

	// check collision before this line - here it will get out of our hands
	if err := findCollisions(cliApp.Commands, os.Args); err != nil {
		term.WriteError(err)
		return 7
	}

	if err := cliApp.RunContext(ctx, os.Args); err != nil {
		return 6
	}

	return 0
}

func findCollisions(availableCmds []*cli.Command, args []string) error {
	// check names and aliases

	// for some built in commands, we need to check their first parameter (args[2])
	metaCmds := []string{"help", "uninstall", "update"}
	for _, c := range metaCmds {
		if c == args[1] && len(args) > 2 {
			if err := findDuplicate(availableCmds, args[2]); err != nil {
				return err
			}

			return nil
		}
	}

	// rest of commands: we need to check the first parameter (args[1])
	if err := findDuplicate(availableCmds, args[1]); err != nil {
		return err
	}

	return nil
}

func findDuplicate(availableCmds []*cli.Command, cmdName string) error {
	matching := make([]string, 0, 2)
	for _, cmd := range availableCmds {
		// match with command name
		if cmd.Name == cmdName {
			matching = append(matching, cmd.Name)
			continue
		}
		// match with command aliases
		for _, alias := range cmd.Aliases {
			if alias == cmdName {
				matching = append(matching, cmdName)
				break
			}
		}

	}

	if len(matching) > 1 {
		return fmt.Errorf("this command is ambiguous, please use prefix of the package which should be used (i.e. custom/command): %s", cmdName)
	}

	return nil
}

func cleanupUpgrade() error {
	filename := filepath.Base(os.Args[0])
	var oldExe string
	if strings.HasSuffix(strings.ToLower(filename), ".exe") {
		oldExe = fmt.Sprintf(".%s.old", filename)
	} else {
		oldExe = fmt.Sprintf(".%s.exe.old", filename)
	}
	return os.Remove(filepath.Join(filepath.Dir(os.Args[0]), oldExe))
}

func checkUpgrade(ctx context.Context) {
	if len(os.Args) > 1 && os.Args[1] == "upgrade" {
		return
	}
	if latestVersion := commands.CheckUpgradeVersion(ctx, false); latestVersion != "" && latestVersion != version.Version {
		if commands.UpgradeCli(ctx, latestVersion) {
			stats.TrackEvent(ctx, "upgrade.auto", "success", "to: "+latestVersion+" from: "+version.Version)
			return
		}
		stats.TrackEvent(ctx, "upgrade.auto", "failed", "to: "+latestVersion+" from: "+version.Version)
	}
}
