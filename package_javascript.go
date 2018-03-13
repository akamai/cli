package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func installJavaScript(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("node")
	if err != nil {
		bin, err = exec.LookPath("nodejs")
		if err != nil {
			return false, cli.NewExitError(color.RedString("Unable to locate Node.js runtime"), 1)
		}
	}

	if cmdPackage.Requirements.Node != "" && cmdPackage.Requirements.Node != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		r, _ := regexp.Compile("^v(.*?)\\s*$")
		matches := r.FindStringSubmatch(string(output))
		if versionCompare(cmdPackage.Requirements.Node, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("Node.js %s is required to install this command.", cmdPackage.Requirements.Node), 1)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "yarn.lock")); err == nil {
		bin, err := exec.LookPath("yarn")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		bin, err := exec.LookPath("npm")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, cli.NewExitError("Unable to find package manager.", 1)
}
