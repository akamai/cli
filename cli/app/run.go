package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/akamai/cli/pkg/app"
	"github.com/akamai/cli/pkg/commands"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/akamai/cli/pkg/version"
	"github.com/fatih/color"
)

// Run ...
func Run() int {
	term := terminal.Color()
	if err := os.Setenv("AKAMAI_CLI", "1"); err != nil {
		term.WriteErrorf("Unable to set AKAMAI_CLI: %s", err.Error())
		return 1
	}
	if err := os.Setenv("AKAMAI_CLI_VERSION", version.Version); err != nil {
		term.WriteErrorf("Unable to set AKAMAI_CLI_VERSION: %s", err.Error())
		return 1
	}

	cachePath := config.GetConfigValue("cli", "cache-path")
	if cachePath == "" {
		cliHome, _ := tools.GetAkamaiCliPath()

		cachePath = filepath.Join(cliHome, "cache")
		if err := os.MkdirAll(cachePath, 0700); err != nil {
			term.WriteErrorf("Unable to create cache directory: %s", err.Error())
			return 2
		}
	}

	ctx := terminal.Context(context.Background(), term)

	config.SetConfigValue("cli", "cache-path", cachePath)
	if err := config.SaveConfig(ctx); err != nil {
		return 3
	}
	if err := config.ExportConfigEnv(ctx); err != nil {
		term.WriteErrorf("Unable to export required envs: %s", err.Error())
	}

	cli := app.CreateApp(ctx)
	ctx = log.SetupContext(ctx, cli.Writer)

	cmds, err := commands.CommandLocator(ctx)
	if err != nil {
		term.WriteError(color.RedString("An error occurred initializing commands"))
		return 4
	}
	cli.Commands = cmds

	if err := firstRun(ctx); err != nil {
		return 5
	}
	checkUpgrade(ctx)
	if err := stats.CheckPing(ctx); err != nil {
		term.WriteError(err.Error())
	}

	if err := cli.RunContext(ctx, os.Args); err != nil {
		return 6
	}

	return 0
}

func checkUpgrade(ctx context.Context) {
	if latestVersion := commands.CheckUpgradeVersion(ctx, false); latestVersion != "" {
		if commands.UpgradeCli(ctx, latestVersion) {
			stats.TrackEvent(ctx, "upgrade.auto", "success", "to: "+latestVersion+" from: "+version.Version)
			return
		}
		stats.TrackEvent(ctx, "upgrade.auto", "failed", "to: "+latestVersion+" from: "+version.Version)
	}
}
