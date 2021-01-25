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

package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/akamai/cli/pkg/app"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/tools"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

type command struct {
	Name         string   `json:"name"`
	Aliases      []string `json:"aliases"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Usage        string   `json:"usage"`
	Arguments    string   `json:"arguments"`
	Bin          string   `json:"bin"`
	AutoComplete bool     `json:"auto-complete"`

	Flags       []cli.Flag     `json:"-"`
	Docs        string         `json:"-"`
	BinSuffix   string         `json:"-"`
	OS          string         `json:"-"`
	Arch        string         `json:"-"`
	Subcommands []*cli.Command `json:"-"`
}

func getBuiltinCommands() []subcommands {
	commands := []subcommands{
		{
			Commands: []command{
				{
					Name:        "config",
					Arguments:   "<action> <setting> [value]",
					Description: "Manage configuration",
					Subcommands: []*cli.Command{
						{
							Name:      "get",
							ArgsUsage: "<setting>",
							Action:    cmdConfigGet,
						},
						{
							Name:      "set",
							ArgsUsage: "<setting> <value>",
							Action:    cmdConfigSet,
						},
						{
							Name:      "list",
							ArgsUsage: "[section]",
							Action:    cmdConfigList,
						},
						{
							Name:      "unset",
							Aliases:   []string{"rm"},
							ArgsUsage: "<setting>",
							Action:    cmdConfigUnset,
						},
					},
				},
			},
		},
		{
			Commands: []command{
				{
					Name:        "help",
					Description: "Displays help information",
					Arguments:   "[command] [sub-command]",
				},
			},
			Action: cmdHelp,
		},
		{
			Commands: []command{
				{
					Name:        "install",
					Arguments:   "<package name or repository URL>...",
					Description: "Fetch and install packages from a Git repository.",
					Flags: []cli.Flag{
						&cli.BoolFlag{
							Name:  "force",
							Usage: "Force binary installation if available when source installation fails",
						},
					},
					Aliases: []string{"get"},
					Docs: fmt.Sprintf("Examples:\n\n   %v\n,  %v\n   %v\n   %v",
						"akamai install property purge",
						"akamai install akamai/cli-property",
						"akamai install git@github.com:akamai/cli-property.git",
						"akamai install https://github.com/akamai/cli-property.git"),
				},
			},
			Action: cmdInstall,
		},
		{
			Commands: []command{
				{
					Name:        "list",
					Description: "Displays available commands",
					Flags: []cli.Flag{
						&cli.BoolFlag{
							Name:  "remote",
							Usage: "Display all available packages",
						},
					},
				},
			},
			Action: cmdList,
		},
		{
			Commands: []command{
				{
					Name:        "search",
					Arguments:   "<keyword>...",
					Description: "Search for packages in the official Akamai CLI package repository",
					Docs:        "Examples:\n\n   akamai search property",
				},
			},
			Action: cmdSearch,
		},
		{
			Commands: []command{
				{
					Name:        "uninstall",
					Arguments:   "<command>...",
					Description: "Uninstall package containing <command>",
				},
			},
			Action: cmdUninstall,
		},
		{
			Commands: []command{
				{
					Name:        "update",
					Arguments:   "[<command>...]",
					Description: "Update one or more commands. If no command is specified, all commands are updated",
					Flags: []cli.Flag{
						&cli.BoolFlag{
							Name:  "force",
							Usage: "Force binary installation if available when source installation fails",
						},
					},
				},
			},
			Action: cmdUpdate,
		},
	}

	upgradeCommand := getUpgradeCommand()
	if upgradeCommand != nil {
		commands = append(commands, *upgradeCommand)
	}

	return commands
}

func getCommands() []subcommands {
	var (
		commandMap   = make(map[string]subcommands)
		commandOrder = make([]string, 0)
		commands     = make([]subcommands, 0)
	)
	for _, pkg := range getBuiltinCommands() {
		for _, command := range pkg.Commands {
			commandMap[command.Name] = pkg
			commandOrder = append(commandOrder, command.Name)
		}
	}

	packagePaths := getPackagePaths()
	for _, dir := range packagePaths {
		pkg, err := readPackage(dir)
		if err == nil {
			for key, command := range pkg.Commands {
				commandPkg := pkg
				commandPkg.Commands = commandPkg.Commands[key : key+1]
				commandMap[command.Name] = commandPkg
				commandOrder = append(commandOrder, command.Name)
			}
		}
	}

	sort.Strings(commandOrder)
	for _, key := range commandOrder {
		commands = append(commands, commandMap[key])
	}

	return commands
}

// CommandLocator ...
func CommandLocator(ctx context.Context) ([]*cli.Command, error) {
	commands := make([]*cli.Command, 0)
	builtinCmds := make(map[string]bool)
	for _, cmd := range getBuiltinCommands() {
		builtinCmds[strings.ToLower(cmd.Commands[0].Name)] = true
		commands = append(
			commands,
			&cli.Command{
				Name:         strings.ToLower(cmd.Commands[0].Name),
				Aliases:      cmd.Commands[0].Aliases,
				Usage:        cmd.Commands[0].Usage,
				ArgsUsage:    cmd.Commands[0].Arguments,
				Description:  cmd.Commands[0].Description,
				Action:       cmd.Action,
				UsageText:    cmd.Commands[0].Docs,
				Flags:        cmd.Commands[0].Flags,
				Subcommands:  cmd.Commands[0].Subcommands,
				HideHelp:     true,
				BashComplete: app.DefaultAutoComplete,
			},
		)
	}

	for _, cmd := range getCommands() {
		for _, command := range cmd.Commands {
			if _, ok := builtinCmds[command.Name]; ok {
				continue
			}

			commands = append(
				commands,
				&cli.Command{
					Name:        strings.ToLower(command.Name),
					Aliases:     command.Aliases,
					Description: command.Description,

					Action:          cmdSubcommand,
					Category:        color.YellowString("Installed Commands:"),
					SkipFlagParsing: true,
					BashComplete: func(c *cli.Context) {
						if command.AutoComplete {
							executable, err := findExec(ctx, c.Command.Name)
							if err != nil {
								return
							}

							executable = append(executable, os.Args[2:]...)
							if err = passthruCommand(executable); err != nil {
								return
							}
						}
					},
				},
			)
		}
	}

	return commands, nil
}

func findExec(ctx context.Context, cmd string) ([]string, error) {
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
	if err := os.Setenv("PATH", packagePaths); err != nil {
		return nil, err
	}

	// Quick look for executables on the path
	var path string
	path, err := exec.LookPath(cmdName)
	if err != nil {
		path, _ = exec.LookPath(cmdNameTitle)
	}

	if path != "" {
		if err := os.Setenv("PATH", systemPath); err != nil {
			return nil, err
		}
		return []string{path}, nil
	}

	if err := os.Setenv("PATH", systemPath); err != nil {
		return nil, err
	}
	if packagePaths == "" {
		return nil, errors.New("no executables found")
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
		case language == languageGO || language == languageC || language == languageCSharp:
			err = nil
			cmd = []string{cmdFile}
		case language == languageJavaScript:
			bin, err = exec.LookPath("node")
			if err != nil {
				bin, err = exec.LookPath("nodejs")
			}
			cmd = []string{bin, cmdFile}
		case language == languagePython:
			var bins packages.PythonBins
			bins, err = packages.FindPythonBins(ctx, cmdPackage.Requirements.Python)
			if err != nil {
				return nil, err
			}
			bin = bins.Python

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

	return nil, errors.New("no executables found")
}

func getPackageBinPaths() string {
	path := ""
	akamaiCliPath, err := tools.GetAkamaiCliSrcPath()
	if err == nil && akamaiCliPath != "" {
		paths, _ := filepath.Glob(filepath.Join(akamaiCliPath, "*"))
		if len(paths) > 0 {
			path += strings.Join(paths, string(os.PathListSeparator))
		}
		paths, _ = filepath.Glob(filepath.Join(akamaiCliPath, "*", "bin"))
		if len(paths) > 0 {
			path += string(os.PathListSeparator) + strings.Join(paths, string(os.PathListSeparator))
		}
	}

	return path
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
		return cli.Exit("", exitCode)
	}
	return nil
}
