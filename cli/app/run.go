package app

import (
	"context"
	"github.com/akamai/cli/pkg/packages"
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
)

// Run ...
func Run() int {
	ctx := context.Background()
	term := terminal.Color()
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

	cli := app.CreateApp(ctx)
	ctx = log.SetupContext(ctx, cli.Writer)

	cmds := commands.CommandLocator(ctx)
	cli.Commands = cmds

	if err := firstRun(ctx); err != nil {
		return 5
	}
	checkUpgrade(ctx, packages.NewLangManager())
	if err := stats.CheckPing(ctx); err != nil {
		term.WriteError(err.Error())
	}

	if err := cli.RunContext(ctx, os.Args); err != nil {
		return 6
	}

	return 0
}

func checkUpgrade(ctx context.Context, langManager packages.LangManager) {
	if latestVersion := commands.CheckUpgradeVersion(ctx, false); latestVersion != "" {
		if commands.UpgradeCli(ctx, latestVersion, langManager) {
			stats.TrackEvent(ctx, "upgrade.auto", "success", "to: "+latestVersion+" from: "+version.Version)
			return
		}
		stats.TrackEvent(ctx, "upgrade.auto", "failed", "to: "+latestVersion+" from: "+version.Version)
	}
}
