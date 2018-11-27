// Copyright 2018. Akamai Technologies, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	akamai "github.com/akamai/cli-common-golang"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli"
)

func self() string {
	return filepath.Base(os.Args[0])
}

func getAkamaiCliPath() (string, error) {
	cliHome := os.Getenv("AKAMAI_CLI_HOME")
	if cliHome == "" {
		var err error
		cliHome, err = homedir.Dir()
		if err != nil {
			return "", cli.NewExitError("Package install directory could not be found. Please set $AKAMAI_CLI_HOME.", -1)
		}
	}

	cliPath := filepath.Join(cliHome, ".akamai-cli")
	err := os.MkdirAll(cliPath, 0700)
	if err != nil {
		return "", cli.NewExitError("Unable to create Akamai CLI root directory.", -1)
	}

	return cliPath, nil
}

func getAkamaiCliSrcPath() (string, error) {
	cliHome, _ := getAkamaiCliPath()

	return filepath.Join(cliHome, "src"), nil
}

func getAkamaiCliCachePath() (string, error) {
	if cachePath := getConfigValue("cli", "cache-path"); cachePath != "" {
		return cachePath, nil
	}

	cliHome, _ := getAkamaiCliPath()

	cachePath := filepath.Join(cliHome, "cache")
	err := os.MkdirAll(cachePath, 0700)
	if err != nil {
		return "", err
	}

	setConfigValue("cli", "cache-path", cachePath)
	saveConfig()

	return cachePath, nil
}

func findExec(cmd string) ([]string, error) {
	// "command" becomes: akamai-command, and akamaiCommand
	// "command-name" becomes: akamai-command-name, and akamaiCommandName
	cmdName := "akamai"
	cmdNameTitle := "akamai"
	for _, cmdPart := range strings.Split(cmd, "-") {
		cmdName += "-" + strings.ToLower(cmdPart)
		cmdNameTitle += strings.Title(strings.ToLower(cmdPart))
	}

	systemPath := os.Getenv("PATH")
	packagePaths := getPackageBinPaths()
	os.Setenv("PATH", packagePaths)

	// Quick look for executables on the path
	var path string
	path, err := exec.LookPath(cmdName)
	if err != nil {
		path, _ = exec.LookPath(cmdNameTitle)
	}

	if path != "" {
		os.Setenv("PATH", systemPath)
		return []string{path}, nil
	}

	os.Setenv("PATH", systemPath)
	if packagePaths == "" {
		return nil, errors.New("No executables found.")
	}

	for _, path := range filepath.SplitList(packagePaths) {
		filePaths := []string{
			// Search for <path>/akamai-command, <path>/akamaiCommand
			filepath.Join(path, cmdName),
			filepath.Join(path, cmdNameTitle),

			// Search for <path>/akamai-command.*, <path>/akamaiCommand.*
			// This should catch .exe, .bat, .com, .cmd, and .jar
			filepath.Join(path, cmdName+".*"),
			filepath.Join(path, cmdNameTitle+".*"),
		}

		var files []string
		for _, filePath := range filePaths {
			files, _ = filepath.Glob(filePath)
			if len(files) > 0 {
				break
			}
		}

		if len(files) == 0 {
			continue
		}

		cmdFile := files[0]

		packageDir := findPackageDir(filepath.Dir(cmdFile))
		cmdPackage, err := readPackage(packageDir)
		if err != nil {
			return nil, err
		}

		language := determineCommandLanguage(cmdPackage)
		var (
			cmd []string
			bin string
		)
		switch {
		// Compiled Languages
		case language == "go" || language == "c#" || language == "csharp":
			err = nil
			cmd = []string{cmdFile}
		case language == "javascript":
			bin, err = exec.LookPath("node")
			if err != nil {
				bin, err = exec.LookPath("nodejs")
			}
			cmd = []string{bin, cmdFile}
		case language == "python":
			var bins pythonBins
			bins, err = findPythonBins(cmdPackage.Requirements.Python)
			bin = bins.python

			cmd = []string{bin, cmdFile}
			// Other languages (php, perl, ruby, etc.)
		default:
			bin, err = exec.LookPath(language)
			cmd = []string{bin, cmdFile}
		}

		if err != nil {
			return nil, err
		}

		return cmd, nil
	}

	return nil, errors.New("No executables found.")
}

func passthruCommand(executable []string) error {
	subCmd := exec.Command(executable[0], executable[1:]...)
	subCmd.Stdin = os.Stdin
	subCmd.Stderr = os.Stderr
	subCmd.Stdout = os.Stdout
	err := subCmd.Run()

	exitCode := 1
	if exitError, ok := err.(*exec.ExitError); ok {
		if waitStatus, ok := exitError.Sys().(syscall.WaitStatus); ok {
			exitCode = waitStatus.ExitStatus()
		}
	}
	if err != nil {
		return cli.NewExitError("", exitCode)
	}
	return nil
}

func githubize(repo string) string {
	if strings.HasPrefix(repo, "http") || strings.HasPrefix(repo, "ssh") || strings.HasSuffix(repo, ".git") {
		return strings.TrimPrefix(repo, "ssh://")
	}

	if strings.HasPrefix(repo, "file://") {
		return repo
	}

	if !strings.Contains(repo, "/") {
		repo = "akamai/cli-" + strings.TrimPrefix(repo, "cli-")
	}

	// Handle Github migration from akamai-open -> akamai
	if strings.HasPrefix(repo, "akamai-open/") {
		repo = "akamai/" + strings.TrimPrefix(repo, "akamai-open/")
	}

	return "https://github.com/" + repo + ".git"
}

func versionCompare(left string, right string) int {
	leftParts := strings.Split(left, ".")
	leftMajor, _ := strconv.Atoi(leftParts[0])
	leftMinor := 0
	leftMicro := 0

	if left == right {
		return 0
	}

	if len(leftParts) > 1 {
		leftMinor, _ = strconv.Atoi(leftParts[1])
	}
	if len(leftParts) > 2 {
		leftMicro, _ = strconv.Atoi(leftParts[2])
	}

	rightParts := strings.Split(right, ".")
	rightMajor, _ := strconv.Atoi(rightParts[0])
	rightMinor := 0
	rightMicro := 0

	if len(rightParts) > 1 {
		rightMinor, _ = strconv.Atoi(rightParts[1])
	}
	if len(rightParts) > 2 {
		rightMicro, _ = strconv.Atoi(rightParts[2])
	}

	if leftMajor > rightMajor {
		return -1
	}

	if leftMajor == rightMajor && leftMinor > rightMinor {
		return -1
	}

	if leftMajor == rightMajor && leftMinor == rightMinor && leftMicro > rightMicro {
		return -1
	}

	return 1
}

func showBanner() {
	fmt.Fprintln(akamai.App.ErrWriter)
	bg := color.New(color.BgMagenta)
	fmt.Fprintf(akamai.App.ErrWriter, bg.Sprintf(strings.Repeat(" ", 60)+"\n"))
	fg := bg.Add(color.FgWhite)
	title := "Welcome to Akamai CLI v" + VERSION
	ws := strings.Repeat(" ", 16)
	fmt.Fprintf(akamai.App.ErrWriter, fg.Sprintf(ws+title+ws+"\n"))
	fmt.Fprintf(akamai.App.ErrWriter, bg.Sprintf(strings.Repeat(" ", 60)+"\n"))
	fmt.Fprintln(akamai.App.ErrWriter)
}
