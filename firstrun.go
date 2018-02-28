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

	bannerShown, err := checkPath()
	if err != nil {
		return err
	}

	bannerShown = checkUpdate(bannerShown)
	checkStats(bannerShown)

	return nil
}

func checkPath() (bool, error) {
	selfPath, err := osext.Executable()
	if err != nil {
		return false, err
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
		inPath = true
		checkUpdate(!inPath)
	}

	if len(paths) == 0 {
		inPath = true
		checkUpdate(!inPath)
	}

	for _, path := range paths {
		if len(strings.TrimSpace(path)) == 0 {
			continue
		}

		if runtime.GOOS == "windows" {
			path = strings.ToLower(path)
		}

		if err := checkAccess(path, ACCESS_W_OK); err == nil {
			writablePaths = append(writablePaths, path)
		}

		if path == dirPath {
			checkUpdate(false)
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
			checkUpdate(true)
		}

		choosePath(writablePaths, answer, selfPath)
	}

	return !inPath, nil
}

func choosePath(writablePaths []string, answer string, selfPath string) {
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
		choosePath(writablePaths, answer, selfPath)
	}
	if answer == "" || index < 1 || index > len(writablePaths) {
		color.Red("Invalid choice, try again")
		choosePath(writablePaths, answer, selfPath)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	newPath := filepath.Join(writablePaths[index-1], "akamai"+suffix)
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

func checkUpdate(bannerShown bool) bool {
	if getConfigValue("cli", "last-upgrade-check") == "" {
		if !bannerShown {
			bannerShown = true
			showBanner()
		}
		fmt.Print("Akamai CLI can auto-update itself, would you like to enable daily checks? [Y/n]: ")

		answer := ""
		fmt.Scanln(&answer)
		if answer != "" && strings.ToLower(answer) != "y" {
			setConfigValue("cli", "last-upgrade-check", "ignore")
			saveConfig()
			return bannerShown
		}

		setConfigValue("cli", "last-upgrade-check", "never")
		saveConfig()
	}

	return bannerShown
}

func checkStats(bannerShown bool) bool {
	if getConfigValue("cli", "client-id") == "" && getConfigValue("cli", "enable-cli-statistics") != "false" {
		if !bannerShown {
			bannerShown = true
			showBanner()
		}
		anonymous := color.New(color.FgWhite, color.Bold).Sprint("anonymous")
		fmt.Printf("Help Akamai improve Akamai CLI by automatically sending %s diagnotics and usage data.\n", anonymous)
		fmt.Println("Examples of data being send include upgrade statistics, and packages installed and updated.\n\n")
		fmt.Printf("Send %s diagnostics and usage data to Akamai? [Y/n]: ", anonymous)

		answer := ""
		fmt.Scanln(&answer)
		if answer != "" && strings.ToLower(answer) != "y" {
			setConfigValue("cli", "enable-cli-statistics", "false")
			saveConfig()
			return bannerShown
		}

		setConfigValue("cli", "enable-cli-statistics", "true")
		setConfigValue("cli", "last-ping", "never")
		setupUuid()
		saveConfig()
		trackEvent("first-run", "true")
	}

	return bannerShown
}
