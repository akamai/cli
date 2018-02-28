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
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/urfave/cli"
)

const (
	VERSION = "0.5.1"
)

func main() {
	os.Setenv("AKAMAI_CLI", "1")

	setCliTemplates()
	getAkamaiCliCachePath()

	exportConfigEnv()

	app := cli.NewApp()
	app.Name = "akamai"
	app.Usage = "Akamai CLI"
	app.Version = VERSION
	app.Copyright = "Copyright (C) Akamai Technologies, Inc"
	app.Writer = colorable.NewColorableStdout()
	app.ErrWriter = colorable.NewColorableStderr()

	firstRun()

	if latestVersion := checkForUpgrade(false); latestVersion != "" {
		if upgradeCli(latestVersion) {
			return
		}
	}

	var builtinCmds map[string]bool = make(map[string]bool)
	for _, cmd := range getBuiltinCommands() {
		builtinCmds[strings.ToLower(cmd.Commands[0].Name)] = true
		app.Commands = append(
			app.Commands,
			cli.Command{
				Name:        strings.ToLower(cmd.Commands[0].Name),
				Aliases:     cmd.Commands[0].Aliases,
				Usage:       cmd.Commands[0].Usage,
				ArgsUsage:   cmd.Commands[0].Arguments,
				Description: cmd.Commands[0].Description,
				Action:      cmd.action,
				UsageText:   cmd.Commands[0].Docs,
				Flags:       cmd.Commands[0].Flags,
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
				},
			)
		}
	}

	app.Run(os.Args)
}