//go:build !noautoupgrade

package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/version"
	"github.com/urfave/cli/v3"
)

func cmdUpgrade(ctx context.Context, cmd *cli.Command) error {
	ctx = log.WithCommandContext(ctx, cmd.Name)
	logger := log.FromContext(ctx)
	start := time.Now()
	logger.Debug("UPGRADE START")
	defer func() {
		logger.Debug(fmt.Sprintf("UPGRADE FINISH: %v", time.Since(start)))
	}()
	term := terminal.Get(ctx)

	term.Spinner().Start("Checking for upgrades...")

	latestVersion := CheckUpgradeVersion(ctx, true)
	if latestVersion != "" && latestVersion != version.Version {
		term.Spinner().Stop(terminal.SpinnerStatusOK)
		os.Args = []string{os.Args[0], "--version"}
		return UpgradeCli(ctx, latestVersion)
	}
	term.Spinner().Stop(terminal.SpinnerStatusWarnOK)
	if latestVersion == version.Version {
		term.Printf("Akamai CLI (%s) is already up-to-date", color.CyanString("v"+version.Version))
		return nil
	}
	if latestVersion != "" {
		term.Printf("Akamai CLI version: %s", color.CyanString("v"+version.Version))
	}
	return nil
}
