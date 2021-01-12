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
	"errors"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/tools"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	akamai "github.com/akamai/cli-common-golang"
	"github.com/fatih/color"
	"github.com/urfave/cli"
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

	Flags       []cli.Flag    `json:"-"`
	Docs        string        `json:"-"`
	BinSuffix   string        `json:"-"`
	OS          string        `json:"-"`
	Arch        string        `json:"-"`
	Subcommands []cli.Command `json:"-"`
}

func packageListDiff(oldcmds []CommandPackage) {
	cmds := getCommands()

	var old []command
	for _, oldcmd := range oldcmds {
		for _, cmd := range oldcmd.Commands {
			old = append(old, cmd)
		}
	}

	var newCmds []command
	for _, newcmd := range cmds {
		for _, cmd := range newcmd.Commands {
			newCmds = append(newCmds, cmd)
		}
	}

	var added = make(map[string]bool)
	var removed = make(map[string]bool)

	for _, newCmd := range newCmds {
		found := false
		for _, oldCmd := range old {
			if newCmd.Name == oldCmd.Name {
				found = true
				break
			}
		}

		if !found {
			added[newCmd.Name] = true
		}
	}

	for _, oldCmd := range old {
		found := false
		for _, newCmd := range newCmds {
			if newCmd.Name == oldCmd.Name {
				found = true
				break
			}
		}

		if !found {
			removed[oldCmd.Name] = true
		}
	}

	listInstalledCommands(added, removed)
}

func getBuiltinCommands() []CommandPackage {
	commands := []CommandPackage{
		{
			Commands: []command{
				{
					Name:        "config",
					Arguments:   "<action> <setting> [value]",
					Description: "Manage configuration",
					Subcommands: []cli.Command{
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
						cli.BoolFlag{
							Name:  "force",
							Usage: "Force binary installation if available when source installation fails",
						},
					},
					Aliases: []string{"get"},
					Docs:    "Examples:\n\n   akamai install property purge\n   akamai install akamai/cli-property\n   akamai install git@github.com:akamai/cli-property.git\n   akamai install https://github.com/akamai/cli-property.git",
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
						cli.BoolFlag{
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
						cli.BoolFlag{
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

func getCommands() []CommandPackage {
	var (
		commandMap   = make(map[string]CommandPackage)
		commandOrder = make([]string, 0)
		commands     = make([]CommandPackage, 0)
	)
	for _, pkg := range getBuiltinCommands() {
		for _, command := range pkg.Commands {
			commandMap[command.Name] = pkg
			commandOrder = append(commandOrder, command.Name)
		}
	}

	packagePaths := GetPackagePaths()
	if len(packagePaths) == 0 {

	}

	for _, dir := range packagePaths {
		pkg, err := ReadPackage(dir)
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

var CommandLocator akamai.CommandLocator = func() ([]cli.Command, error) {
	commands := make([]cli.Command, 0)
	builtinCmds := make(map[string]bool)
	for _, cmd := range getBuiltinCommands() {
		builtinCmds[strings.ToLower(cmd.Commands[0].Name)] = true
		commands = append(
			commands,
			cli.Command{
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
				BashComplete: akamai.DefaultAutoComplete,
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
				cli.Command{
					Name:        strings.ToLower(command.Name),
					Aliases:     command.Aliases,
					Description: command.Description,

					Action:          cmdSubcommand,
					Category:        color.YellowString("Installed Commands:"),
					SkipFlagParsing: true,
					BashComplete: func(c *cli.Context) {
						if command.AutoComplete {
							executable, err := findExec(c.Command.Name)
							if err != nil {
								return
							}

							executable = append(executable, os.Args[2:]...)
							tools.PassthruCommand(executable)
						}
					},
				},
			)
		}
	}

	return commands, nil
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

		packageDir := FindPackageDir(filepath.Dir(cmdFile))
		cmdPackage, err := ReadPackage(packageDir)
		if err != nil {
			return nil, err
		}

		language := DetermineCommandLanguage(cmdPackage)
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
			var bins packages.PythonBins
			bins, err = packages.FindPythonBins(cmdPackage.Requirements.Python)
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

	return nil, errors.New("No executables found.")
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
