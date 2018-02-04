package main

import (
	"os"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func cmdSubcommand(c *cli.Context) error {
	cmd := c.Command.Name

	executable, err := findExec(cmd)
	if err != nil {
		return cli.NewExitError(color.RedString("Executable \"%s\" not found.", cmd), 1)
	}

	var packageDir string
	if len(executable) == 1 {
		packageDir = findPackageDir(executable[0])
	} else if len(executable) > 1 {
		packageDir = findPackageDir(executable[1])
	}

	cmdPackage, _ := readPackage(packageDir)

	if cmdPackage.Requirements.Python != "" {
		os.Setenv("PYTHONUSERBASE", packageDir)
		if err != nil {
			return err
		}
	}

	executable = append(executable, os.Args[2:]...)
	return passthruCommand(executable)
}
