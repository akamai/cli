package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func cmdUninstall(c *cli.Context) error {
	for _, cmd := range c.Args() {
		if err := uninstallPackage(cmd); err != nil {
			return err
		}
	}

	return nil
}

func uninstallPackage(cmd string) error {
	exec, err := findExec(cmd)
	if err != nil {
		return cli.NewExitError(color.RedString("Command \"%s\" not found. Try \"%s help\".\n", cmd, self()), 1)
	}

	status := getSpinner(fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd), fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd)+"... ["+color.GreenString("OK")+"]\n")
	status.Start()

	var repoDir string
	if len(exec) == 1 {
		repoDir = findPackageDir(filepath.Dir(exec[0]))
	} else if len(exec) > 1 {
		repoDir = findPackageDir(filepath.Dir(exec[len(exec)-1]))
	}

	if repoDir == "" {
		status.FinalMSG = fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd) + "... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		return cli.NewExitError(color.RedString("unable to uninstall, was it installed using "+color.CyanString("\"akamai install\"")+"?"), 1)
	}

	if err := os.RemoveAll(repoDir); err != nil {
		status.FinalMSG = fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd) + "... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		return cli.NewExitError(color.RedString("unable to remove directory: %s", repoDir), 1)
	}

	status.Stop()

	return nil
}
