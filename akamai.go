/*
 * Copyright 2017 Akamai Technologies, Inc
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/kardianos/osext"
	"github.com/mattn/go-colorable"
	"github.com/urfave/cli"
)

const (
	VERSION = "0.6.0"
)

var (
	app *cli.App
)

func main() {
	os.Setenv("AKAMAI_CLI", "1")

	setHelpTemplates()
	getAkamaiCliCachePath()

	exportConfigEnv()

	app = createApp()

	firstRun()

	if latestVersion := checkForUpgrade(false); latestVersion != "" {
		if upgradeCli(latestVersion) {
			trackEvent("upgrade.auto.success", "to: "+latestVersion+" from:"+VERSION)
			return
		} else {
			trackEvent("upgrade.auto.failed", "to: "+latestVersion+" from:"+VERSION)
		}
	}

	checkPing()

	var builtinCmds map[string]bool = make(map[string]bool)
	for _, cmd := range getBuiltinCommands() {
		builtinCmds[strings.ToLower(cmd.Commands[0].Name)] = true
		app.Commands = append(
			app.Commands,
			cli.Command{
				Name:         strings.ToLower(cmd.Commands[0].Name),
				Aliases:      cmd.Commands[0].Aliases,
				Usage:        cmd.Commands[0].Usage,
				ArgsUsage:    cmd.Commands[0].Arguments,
				Description:  cmd.Commands[0].Description,
				Action:       cmd.action,
				UsageText:    cmd.Commands[0].Docs,
				Flags:        cmd.Commands[0].Flags,
				Subcommands:  cmd.Commands[0].Subcommands,
				HideHelp:     true,
				BashComplete: DefaultAutoComplete,
			},
		)
	}

	for _, cmd := range getCommands() {
		for _, command := range cmd.Commands {
			if _, ok := builtinCmds[command.Name]; ok {
				continue
			}

			app.Commands = append(
				app.Commands,
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
							passthruCommand(executable)
						}
					},
				},
			)
		}
	}

	app.Run(os.Args)
}

func createApp() *cli.App {
	app := cli.NewApp()
	app.Name = "akamai"
	app.Usage = "Akamai CLI"
	app.Version = VERSION
	app.Copyright = "Copyright (C) Akamai Technologies, Inc"
	app.Writer = colorable.NewColorableStdout()
	app.ErrWriter = colorable.NewColorableStderr()
	app.EnableBashCompletion = true
	app.BashComplete = DefaultAutoComplete
	cli.BashCompletionFlag = cli.BoolFlag{
		Name:   "generate-auto-complete",
		Hidden: true,
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "bash",
			Usage: "Output bash auto-complete",
		},
		cli.BoolFlag{
			Name:  "zsh",
			Usage: "Output zsh auto-complete",
		},
	}
	app.Action = func(c *cli.Context) {
		defaultAction(c, app)
	}
	return app
}

func defaultAction(c *cli.Context, app *cli.App) {
	cmd, err := osext.Executable()
	if err != nil {
		cmd = self()
	}

	zshScript := `set -k
# To enable zsh auto-completion, run: eval "$(` + cmd + ` --zsh)"
# We recommend adding this to your .zshrc file
autoload -U compinit && compinit
autoload -U bashcompinit && bashcompinit`

	bashComments := `# To enable bash auto-completion, run: eval "$(` + cmd + ` --bash)"
# We recommend adding this to your .bashrc or .bash_profile file`

	bashScript := `_akamai_cli_bash_autocomplete() {
    local cur opts base
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-auto-complete )
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

complete -F _akamai_cli_bash_autocomplete ` + self()

	if c.Bool("bash") {
		fmt.Fprintln(app.Writer, bashComments)
		fmt.Fprintln(app.Writer, bashScript)
		return
	}

	if c.Bool("zsh") {
		fmt.Fprintln(app.Writer, zshScript)
		fmt.Fprintln(app.Writer, bashScript)
		return
	}

	cli.ShowAppHelpAndExit(c, 0)
}
