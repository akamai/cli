package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func cmdUpgrade(c *cli.Context) error {
	status := getSpinner("Checking for upgrades...", "Checking for upgrades...... ["+color.GreenString("OK")+"]\n")

	status.Start()
	if latestVersion := checkForUpgrade(true); latestVersion != "" {
		status.Stop()
		fmt.Printf("Found new version: %s (current version: %s)\n", color.BlueString("v"+latestVersion), color.BlueString("v"+VERSION))
		os.Args = []string{os.Args[0], "--version"}
		upgradeCli(latestVersion)
	} else {
		status.FinalMSG = "Checking for upgrades...... [" + color.CyanString("OK") + "]\n"
		status.Stop()
		fmt.Printf("Akamai CLI (%s) is already up-to-date", color.CyanString("v"+VERSION))
	}

	return nil
}
