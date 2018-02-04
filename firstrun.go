//+build !nofirstrun

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/kardianos/osext"
	"github.com/mattn/go-isatty"
)

func firstRun() error {
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return nil
	}

	selfPath, err := osext.Executable()
	if err != nil {
		return err
	}
	os.Args[0] = selfPath
	dirPath := filepath.Dir(selfPath)

	if runtime.GOOS == "windows" {
		dirPath = strings.ToLower(dirPath)
	}

	sysPath := os.Getenv("PATH")
	paths := filepath.SplitList(sysPath)
	inPath := false
	writablePaths := []string{}

	if getConfigValue("cli", "install-in-path") == "no" {
		goto checkUpdate
	}

	if len(paths) == 0 {
		goto checkUpdate
	}

	for _, path := range paths {
		if len(strings.TrimSpace(path)) == 0 {
			continue
		}

		if runtime.GOOS == "windows" {
			path = strings.ToLower(path)
		}

		if checkAccess(path, ACCESS_W_OK) != nil {
			writablePaths = append(writablePaths, path)
		}

		if path == dirPath {
			inPath = true
			goto checkUpdate
		}
	}

	if !inPath && len(writablePaths) > 0 {
		showBanner()
		fmt.Print("Akamai CLI is not installed in your PATH, would you like to install it? [Y/n]: ")
		answer := ""
		fmt.Scanln(&answer)
		if answer != "" && strings.ToLower(answer) != "y" {
			setConfigValue("cli", "install-in-path", "no")
			saveConfig()
			goto checkUpdate
		}

	choosePath:

		color.Yellow("Choose where you would like to install Akamai CLI:")

		for i, path := range writablePaths {
			fmt.Printf("(%d) %s\n", i+1, path)
		}

		fmt.Print("Enter a number: ")
		answer = ""
		fmt.Scanln(&answer)
		index, err := strconv.Atoi(answer)
		if err != nil {
			color.Red("Invalid choice, try again")
			goto choosePath
		}

		if answer == "" || index < 1 || index > len(writablePaths) {
			color.Red("Invalid choice, try again")
			goto choosePath
		}

		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}

		newPath := writablePaths[index-1] + string(os.PathSeparator) + "akamai" + suffix

		status := getSpinner(
			"Installing to "+newPath+"...",
			"Installing to "+newPath+"...... ["+color.GreenString("OK")+"]\n",
		)
		status.Start()

		err = os.Rename(selfPath, newPath)
		os.Args[0] = newPath

		if err != nil {
			status.FinalMSG = "Installing to " + newPath + "...... [" + color.RedString("FAIL") + "]\n"
			status.Stop()
			color.Red(err.Error())
		}
		status.Stop()
	}

checkUpdate:

	if getConfigValue("cli", "last-update-check") == "" {
		if inPath {
			showBanner()
		}
		fmt.Print("Akamai CLI can auto-update itself, would you like to enable daily checks? [Y/n]: ")

		answer := ""
		fmt.Scanln(&answer)
		if answer != "" && strings.ToLower(answer) != "y" {
			setConfigValue("cli", "last-update-check", "ignore")

			return nil
		}

		setConfigValue("cli", "last-update-check", "never")
	}

	return nil
}
