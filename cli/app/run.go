package app

import (
	"context"
	"fmt"
	"github.com/akamai/cli/pkg/app"
	"github.com/akamai/cli/pkg/commands"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/tools"
	"github.com/akamai/cli/pkg/version"
	"github.com/fatih/color"
	"os"
	"path/filepath"
)

func Run() int {
	os.Setenv("AKAMAI_CLI", "1")
	os.Setenv("AKAMAI_CLI_VERSION", version.Version)

	cachePath := config.GetConfigValue("cli", "cache-path")
	if cachePath == "" {
		cliHome, _ := tools.GetAkamaiCliPath()

		cachePath = filepath.Join(cliHome, "cache")
		err := os.MkdirAll(cachePath, 0700)
		if err != nil {
			return 1
		}
	}

	config.SetConfigValue("cli", "cache-path", cachePath)
	config.SaveConfig()
	config.ExportConfigEnv()

	// TODO return value should be used once App singleton is removed
	_ = app.CreateApp()
	ctx := log.SetupContext(context.Background(), app.App)
	cmds, err := commands.CommandLocator(ctx)
	if err != nil {
		fmt.Fprintln(app.App.ErrWriter, color.RedString("An error occurred initializing commands"))
		return 2
	}
	app.App.Commands = cmds

	if err := firstRun(); err != nil {
		return 3
	}
	checkUpgrade()
	stats.CheckPing()
	if err := app.App.RunContext(ctx, os.Args); err != nil {
		return 4
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
