package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli"
)

func installGolang(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("go")
	if err != nil {
		return false, cli.NewExitError("Unable to locate Go runtime", 1)
	}

	if cmdPackage.Requirements.Go != "" && cmdPackage.Requirements.Go != "*" {
		cmd := exec.Command(bin, "version")
		output, _ := cmd.Output()
		r, _ := regexp.Compile("go version go(.*?) .*")
		matches := r.FindStringSubmatch(string(output))
		if versionCompare(cmdPackage.Requirements.Go, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("Go %s is required to install this command.", cmdPackage.Requirements.Go), 1)
		}
	}

	goPath, err := homedir.Dir()
	if err != nil {
		return false, cli.NewExitError(color.RedString("Unable to determine home directory"), 1)
	}
	goPath = filepath.Join(goPath, ".akamai-cli")
	os.Setenv("GOPATH", os.Getenv("GOPATH")+string(os.PathListSeparator)+goPath)

	if _, err := os.Stat(filepath.Join(dir, "glide.lock")); err == nil {
		bin, err := exec.LookPath("glide")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, cli.NewExitError(err.Error(), 1)
			}
		} else {
			return false, cli.NewExitError("Unable to find package manager.", 1)
		}
	}

	execName := "akamai-" + strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(filepath.Base(dir), "akamai-"), "cli-"))

	cmd := exec.Command(bin, "build", "-o", execName, ".")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		return false, cli.NewExitError(err.Error(), 1)
	}

	return true, nil
}
