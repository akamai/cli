//go:build noautoupgrade

package commands

import (
	"context"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/urfave/cli/v2"
)

func cmdUpgrade(c *cli.Context) error {
	return cli.Exit(color.RedString("[WARNING] Upgrade command is not available for your installation. If you installed Akamai CLI with Homebrew, please run 'brew upgrade akamai' in order to perform upgrade."), 1)
}

func CheckUpgradeVersion(_ context.Context, _ bool) string {
	return ""
}
