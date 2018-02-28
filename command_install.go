package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli"
	"gopkg.in/src-d/go-git.v4"
)

func cmdInstall(c *cli.Context) error {
	if !c.Args().Present() {
		return cli.NewExitError(color.RedString("You must specify a repository URL"), 1)
	}

	oldCmds := getCommands()

	for _, repo := range c.Args() {
		err := installPackage(repo, c.Bool("force"))
		if err != nil {
			return err
		}
	}

	packageListDiff(oldCmds)

	return nil
}

func installPackage(repo string, forceBinary bool) error {
	srcPath, err := getAkamaiCliSrcPath()
	if err != nil {
		return err
	}

	_ = os.MkdirAll(srcPath, 0775)

	repo = githubize(repo)

	status := getSpinner(fmt.Sprintf("Attempting to fetch command from %s...", repo), fmt.Sprintf("Attempting to fetch command from %s...", repo)+"... ["+color.GreenString("OK")+"]\n")
	status.Start()

	dirName := strings.TrimSuffix(filepath.Base(repo), ".git")
	packageDir := filepath.Join(srcPath, dirName)
	if _, err := os.Stat(packageDir); err == nil {
		status.FinalMSG = status.Prefix + "... [" + color.RedString("FAIL") + "]\n"
		status.Stop()

		return cli.NewExitError(color.RedString("Package directory already exists (%s)", packageDir), 1)
	}

	_, err = git.PlainClone(packageDir, false, &git.CloneOptions{
		URL:      repo,
		Progress: nil,
	})

	if err != nil {
		os.RemoveAll(packageDir)

		status.FinalMSG = status.Prefix + "... [" + color.RedString("FAIL") + "]\n"
		status.Stop()

		return cli.NewExitError(color.RedString("Unable to clone repository: "+err.Error()), 1)
	}

	status.Stop()

	if strings.HasPrefix(repo, "https://github.com/akamai/cli-") != true && strings.HasPrefix(repo, "git@github.com:akamai/cli-") != true {
		color.Cyan("Disclaimer: You are installing a third-party package, subject to its own terms and conditions. Akamai makes no warranty or representation with respect to the third-party package.")
	}

	if !installPackageDependencies(packageDir, forceBinary) {
		os.RemoveAll(packageDir)
		return cli.NewExitError("", 1)
	}

	return nil
}

func installPackageDependencies(dir string, forceBinary bool) bool {
	status := getSpinner("Installing...", "Installing...... ["+color.GreenString("OK")+"]\n")

	status.Start()
	cmdPackage, err := readPackage(dir)

	if err != nil {
		status.FinalMSG = "Installing...... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		fmt.Println(err.Error())
		return false
	}

	lang := determineCommandLanguage(cmdPackage)

	var success bool
	switch lang {
	case "php":
		success, err = installPHP(dir, cmdPackage)
	case "javascript":
		success, err = installJavaScript(dir, cmdPackage)
	case "ruby":
		success, err = installRuby(dir, cmdPackage)
	case "python":
		success, err = installPython(dir, cmdPackage)
	case "go":
		success, err = installGolang(dir, cmdPackage)
	default:
		status.FinalMSG = "Installing...... [" + color.CyanString("OK") + "]\n"
		status.Stop()
		color.Cyan("Package installed successfully, however package type is unknown, and may or may not function correctly.")
		return true
	}

	if success && err == nil {
		status.Stop()
		return true
	}

	first := true
	for _, cmd := range cmdPackage.Commands {
		if cmd.Bin != "" {
			if first {
				first = false
				status.FinalMSG = "Installing...... [" + color.CyanString("WARN") + "]\n"
				status.Stop()
				color.Cyan(err.Error())
				if !forceBinary {
					if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
						return false
					}

					fmt.Print("Binary command(s) found, would you like to try download and install it? (Y/n): ")
					answer := ""
					fmt.Scanln(&answer)
					if answer != "" && strings.ToLower(answer) != "y" {
						return false
					}
				}

				os.MkdirAll(filepath.Join(dir, "bin"), 0775)
			}

			status := getSpinner("Downloading binary...", "Downloading binary...... ["+color.GreenString("OK")+"]\n")
			status.Start()
			if downloadBin(filepath.Join(dir, "bin"), cmd) {
				status.Stop()
				return true
			} else {
				status.FinalMSG = "Downloading binary...... [" + color.RedString("FAIL") + "]\n"
				status.Stop()
				color.Red("Unable to download binary: " + err.Error())
				return false
			}
		} else {
			if first {
				first = false
				status.FinalMSG = "Installing...... [" + color.RedString("FAIL") + "]\n"
				status.Stop()
				color.Red(err.Error())
				return false
			}
		}
	}

	return true
}
