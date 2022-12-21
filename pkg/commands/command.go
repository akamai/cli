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

	"github.com/akamai/cli/pkg/apphelp"
	"github.com/akamai/cli/pkg/autocomplete"
	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/tools"
	"github.com/akamai/cli/pkg/version"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

type (
	command struct {
		Name         string   `json:"name"`
		Aliases      []string `json:"aliases"`
		Version      string   `json:"version"`
		Description  string   `json:"description"`
		Usage        string   `json:"usage"`
		Arguments    string   `json:"arguments"`
		Bin          string   `json:"bin"`
		AutoComplete bool     `json:"auto-complete"`
		LdFlags      string   `json:"ldflags"`

		Flags       []cli.Flag     `json:"-"`
		Docs        string         `json:"-"`
		BinSuffix   string         `json:"-"`
		OS          string         `json:"-"`
		Arch        string         `json:"-"`
		Subcommands []*cli.Command `json:"-"`
	}

	// Command represents an external command being prepared or run
	Command struct {
		cmd *exec.Cmd
	}

	// Cmd is a wrapper for exec.Cmd methods
	Cmd interface {
		Run() error
	}
)

func getBuiltinCommands(c *cli.Context) []subcommands {
	commands := make([]subcommands, 0)
	for _, cmd := range c.App.Commands {
		// builtin commands do not have Category set
		if cmd.Category != "" {
			continue
		}
		commands = append(commands, cliCommandToSubcommand(cmd))
	}
	return commands
}

func getCommands(c *cli.Context) []subcommands {
	commands := make([]subcommands, 0)
	for _, cmd := range c.App.Commands {
		commands = append(commands, cliCommandToSubcommand(cmd))
	}
	return commands
}

func cliCommandToSubcommand(from *cli.Command) subcommands {
	return subcommands{
		Commands: []command{
			{
				Name:        from.Name,
				Aliases:     from.Aliases,
				Description: from.Description,
				Usage:       from.Usage,
				Arguments:   from.ArgsUsage,
				Flags:       from.Flags,
				Docs:        from.UsageText,
				Subcommands: from.Subcommands,
			},
		},
		Action: from.Action,
	}
}

func subcommandToCliCommands(from subcommands, gitRepo git.Repository, langManager packages.LangManager) []*cli.Command {
	commands := make([]*cli.Command, 0)
	for key, command := range from.Commands {
		commandPkg := from
		commandPkg.Commands = commandPkg.Commands[key : key+1]
		aliases := append(command.Aliases, fmt.Sprintf("%s/%s", from.Pkg, command.Name))

		commands = append(commands, &cli.Command{
			Name:        strings.ToLower(command.Name),
			Aliases:     aliases,
			Description: command.Description,

			Action:          cmdSubcommand(gitRepo, langManager),
			Category:        color.YellowString("Installed Commands:"),
			SkipFlagParsing: true,
			BashComplete: func(c *cli.Context) {
				if command.AutoComplete {
					executable, packageReqs, err := findExec(c.Context, langManager, c.Command.Name)
					if err != nil {
						return
					}

					packageDir, err := findBinPackageDir(executable)
					if err != nil {
						return
					}

					executable = append(executable, os.Args[2:]...)
					subCmd := createCommand(executable[0], executable[1:])
					if err = passthruCommand(c.Context, subCmd, langManager, *packageReqs, packageDir); err != nil {
						return
					}
				}
			},
		})
	}
	return commands
}

// CommandLocator builds a sorted slice of built-in and installed commands
func CommandLocator(ctx context.Context) []*cli.Command {
	gitRepo := git.NewRepository()
	langManager := packages.NewLangManager()
	commands := createBuiltinCommands()
	commands = append(commands, createInstalledCommands(ctx, gitRepo, langManager)...)

	sortCommands(commands)
	return commands
}

func sortCommands(commands []*cli.Command) {
	sort.Slice(commands, func(i, j int) bool {
		cmp := strings.Compare(commands[i].Name, commands[j].Name)
		return cmp < 0
	})
}

func createBuiltinCommands() []*cli.Command {
	gitRepo := git.NewRepository()
	langManager := packages.NewLangManager()
	return []*cli.Command{
		{
			Name:        "config",
			ArgsUsage:   "<action> <setting> [value]",
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
			HideHelp:     true,
			BashComplete: autocomplete.Default,
		},
		{
			Name:        "install",
			Aliases:     []string{"get"},
			ArgsUsage:   "<package name or repository URL>...",
			Description: "Fetch and install packages from a Git repository",
			Action:      cmdInstall(gitRepo, langManager),
			UsageText: fmt.Sprintf("Examples:\n\n   %v\n,  %v\n   %v\n   %v",
				"akamai install property purge",
				"akamai install akamai/cli-property",
				"akamai install git@github.com:akamai/cli-property.git",
				"akamai install https://github.com/akamai/cli-property.git"),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "force",
					Usage: "Force binary installation if available when source installation fails",
				},
			},
			HideHelp:     true,
			BashComplete: autocomplete.Default,
		},
		{
			Name:        "list",
			Description: "By default, displays installed commands. Optionally, can display package commands from Git repositories",
			Action:      cmdList,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "remote",
					Usage: "Display all available packages",
				},
			},
			HideHelp:           true,
			BashComplete:       autocomplete.Default,
			CustomHelpTemplate: apphelp.SimplifiedHelpTemplate,
		},
		{
			Name:         "search",
			ArgsUsage:    "<keyword>...",
			Description:  "Search for packages in the official Akamai CLI package repository",
			Action:       cmdSearch,
			UsageText:    "Examples:\n\n   akamai search property",
			HideHelp:     true,
			BashComplete: autocomplete.Default,
		},
		{
			Name:         "uninstall",
			ArgsUsage:    "<command>...",
			Description:  "Uninstall package containing <command>",
			Action:       cmdUninstall(langManager),
			HideHelp:     true,
			BashComplete: autocomplete.Default,
		},
		{
			Name:        "update",
			ArgsUsage:   "[<command>...]",
			Description: "Update one or more commands. If no command is specified, all commands are updated",
			Action:      cmdUpdate(gitRepo, langManager),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "force",
					Usage: "Force binary installation if available when source installation fails",
				},
			},
			HideHelp:     true,
			BashComplete: autocomplete.Default,
		},
		{
			Name:         "upgrade",
			Description:  "Upgrade Akamai CLI to the latest version",
			Action:       cmdUpgrade,
			BashComplete: autocomplete.Default,
		},
	}
}

func createInstalledCommands(_ context.Context, gitRepo git.Repository, langManager packages.LangManager) []*cli.Command {
	commands := make([]*cli.Command, 0)
	packagePaths := getPackagePaths()
	for _, dir := range packagePaths {
		pkg, err := readPackage(dir)
		if err == nil {
			commands = append(commands, subcommandToCliCommands(pkg, gitRepo, langManager)...)
		}
	}
	return commands
}

// findExec returns paths to language interpreter (if necessary) and package binary
func findExec(ctx context.Context, langManager packages.LangManager, cmd string) ([]string, *packages.LanguageRequirements, error) {
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
		return nil, nil, err
	}

	// Quick look for executables on the path
	var path string
	path, err := exec.LookPath(cmdName)
	if err != nil {
		path, _ = exec.LookPath(cmdNameTitle)
	}

	if path != "" {
		if err := os.Setenv("PATH", systemPath); err != nil {
			return nil, nil, err
		}
		return []string{path}, &packages.LanguageRequirements{}, nil
	}

	if err := os.Setenv("PATH", systemPath); err != nil {
		return nil, nil, err
	}
	if packagePaths == "" {
		return nil, nil, packages.ErrNoExeFound
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

		cmdBinary := files[0]

		packageDir := findPackageDir(filepath.Dir(cmdBinary))
		cmdPackage, err := readPackage(packageDir)
		if err != nil {
			return nil, nil, err
		}

		comm, err := langManager.FindExec(ctx, cmdPackage.Requirements, cmdBinary)
		if err != nil {
			return nil, nil, err
		}

		return comm, &cmdPackage.Requirements, nil
	}

	return nil, nil, packages.ErrNoExeFound
}

func getPackageBinPaths() string {
	path := ""
	if akamaiCliPath, err := tools.GetAkamaiCliSrcPath(); err == nil {
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

// passthruCommand performs the external Cmd invocation and previous set up, if required
func passthruCommand(ctx context.Context, subCmd Cmd, langManager packages.LangManager, languageRequirements packages.LanguageRequirements, dirName string) error {
	/*
		Checking if any additional VE set up for Python commands is required.
		Just run package setup if both conditions below are met:
		* required python >= 3.0.0
		* virtual environment is not present
	*/
	if v3Comparison := version.Compare(languageRequirements.Python, "3.0.0"); languageRequirements.Python != "" && (v3Comparison == version.Greater || v3Comparison == version.Equals) {
		vePath, err := tools.GetPkgVenvPath(filepath.Base(dirName))
		if err != nil {
			return err
		}
		veExists, err := langManager.FileExists(vePath)
		if err != nil {
			return err
		}
		if !veExists {
			if err := langManager.PrepareExecution(ctx, languageRequirements, dirName); err != nil {
				return err
			}
		}
	}
	defer langManager.FinishExecution(ctx, languageRequirements, dirName)

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

func findBinPackageDir(binPath []string) (string, error) {
	if len(binPath) == 0 {
		return "", packages.ErrPackageExecutableNotFound
	}

	absPath, err := filepath.Abs(binPath[len(binPath)-1])
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("the specified binary is a directory")
	}

	return filepath.Dir(filepath.Dir(absPath)), nil
}

func createCommand(name string, args []string) *Command {
	comm := &Command{cmd: exec.Command(name, args...)}
	comm.cmd.Stdin = os.Stdin
	comm.cmd.Stderr = os.Stderr
	comm.cmd.Stdout = os.Stdout

	return comm
}

// Run starts the command and waits for it to complete
func (c *Command) Run() error {
	return c.cmd.Run()
}
