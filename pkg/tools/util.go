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

package tools

import (
	"fmt"
	"github.com/akamai/cli/pkg/version"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	akamai "github.com/akamai/cli-common-golang"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli"
)

func Self() string {
	return filepath.Base(os.Args[0])
}

func GetAkamaiCliPath() (string, error) {
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

func GetAkamaiCliSrcPath() (string, error) {
	cliHome, _ := GetAkamaiCliPath()

	return filepath.Join(cliHome, "src"), nil
}

func PassthruCommand(executable []string) error {
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

func Githubize(repo string) string {
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

func ShowBanner() {
	fmt.Fprintln(akamai.App.ErrWriter)
	bg := color.New(color.BgMagenta)
	fmt.Fprintf(akamai.App.ErrWriter, bg.Sprintf(strings.Repeat(" ", 60)+"\n"))
	fg := bg.Add(color.FgWhite)
	title := "Welcome to Akamai CLI v" + version.Version
	ws := strings.Repeat(" ", 16)
	fmt.Fprintf(akamai.App.ErrWriter, fg.Sprintf(ws+title+ws+"\n"))
	fmt.Fprintf(akamai.App.ErrWriter, bg.Sprintf(strings.Repeat(" ", 60)+"\n"))
	fmt.Fprintln(akamai.App.ErrWriter)
}

// We must copy+unlink the file because moving files is broken across filesystems
func MoveFile(src string, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)

	if err != nil {
		return err
	}

	err = os.Chmod(dst, 0755)
	if err != nil {
		return err
	}

	err = os.Remove(src)
	return err
}
