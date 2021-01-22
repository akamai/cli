package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"

	"github.com/akamai/cli/pkg/app"
	"github.com/akamai/cli/pkg/commands"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/tools"
	"github.com/akamai/cli/pkg/version"
)

// Run ...
func Run() int {
	if err := os.Setenv("AKAMAI_CLI", "1"); err != nil {
		return 1
	}
	if err := os.Setenv("AKAMAI_CLI_VERSION", version.Version); err != nil {
		return 1
	}

	cachePath := config.GetConfigValue("cli", "cache-path")
	if cachePath == "" {
		cliHome, _ := tools.GetAkamaiCliPath()

		cachePath = filepath.Join(cliHome, "cache")
		err := os.MkdirAll(cachePath, 0700)
		if err != nil {
			return 2
		}
	}

	config.SetConfigValue("cli", "cache-path", cachePath)
	if err := config.SaveConfig(); err != nil {
		return 3
	}
	config.ExportConfigEnv()

	// TODO return value should be used once App singleton is removed
	_ = app.CreateApp()
	ctx := log.SetupContext(context.Background(), app.App.Writer)
	cmds, err := commands.CommandLocator(ctx)
	if err != nil {
		fmt.Fprintln(app.App.ErrWriter, color.RedString("An error occurred initializing commands"))
		return 4
	}
	app.App.Commands = cmds

	if err := firstRun(); err != nil {
		return 5
	}
	checkUpgrade()
	stats.CheckPing()
	if err := app.App.RunContext(ctx, os.Args); err != nil {
		return 6
	}

	return 0
}

func checkUpgrade() {
	if latestVersion := commands.CheckUpgradeVersion(false); latestVersion != "" {
		if commands.UpgradeCli(latestVersion) {
			stats.TrackEvent("upgrade.auto", "success", "to: "+latestVersion+" from: "+version.Version)
			return
		}
		stats.TrackEvent("upgrade.auto", "failed", "to: "+latestVersion+" from: "+version.Version)
	}
}
