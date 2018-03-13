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

func installRuby(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("ruby")
	if err != nil {
		return false, cli.NewExitError(color.RedString("Unable to locate Ruby runtime"), 1)
	}

	if cmdPackage.Requirements.Ruby != "" && cmdPackage.Requirements.Ruby != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		r, _ := regexp.Compile("^ruby (.*?)(p.*?) (.*)")
		matches := r.FindStringSubmatch(string(output))
		if versionCompare(cmdPackage.Requirements.Ruby, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("Ruby %s is required to install this command.", cmdPackage.Requirements.Ruby), 1)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "Gemfile")); err == nil {
		bin, err := exec.LookPath("bundle")
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
