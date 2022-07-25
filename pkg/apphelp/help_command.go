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

package apphelp

import (
	"os"

	"github.com/urfave/cli/v2"
)

func cmdHelp(c *cli.Context) error {
	if c.Args().Present() {
		cmdName := c.Args().First()
		cmd := c.App.Command(cmdName)
		if cmd == nil {
			return cli.ShowAppHelp(c)
		}

		if subCmd := c.Args().Get(1); subCmd != "" || len(cmd.Subcommands) > 0 {
			os.Args = append([]string{os.Args[0], cmdName}, c.Args().Tail()...)
			os.Args = append(os.Args, "--help")
			return c.App.Run(os.Args)
		}

		if isBuiltinCommand(c, cmdName) {
			if shouldAddHelpFlag(cmd) {
				cmd.Flags = append(cmd.Flags, cli.HelpFlag)
			}
			return cli.ShowCommandHelp(c, cmdName)
		}

		os.Args = append([]string{os.Args[0], cmdName, "help"}, c.Args().Tail()...)
		return c.App.RunContext(c.Context, os.Args)
	}

	return cli.ShowAppHelp(c)
}

func hasHelpFlag(cmd *cli.Command) bool {
	for _, f := range cmd.Flags {
		if f.Names()[0] == "help" {
			return true
		}
	}
	return false
}

func shouldAddHelpFlag(cmd *cli.Command) bool {
	if !cmd.HideHelp && cli.HelpFlag != nil && !hasHelpFlag(cmd) {
		return true
	}
	return false
}

func isBuiltinCommand(c *cli.Context, cmdName string) bool {
	for _, cmd := range c.App.Commands {
		if cmd.Category != "" {
			continue
		}
		if cmd.Name == cmdName {
			return true
		}
		if contains(cmd.Aliases, cmdName) {
			return true
		}
	}

	return false
}

func contains(slc []string, e string) bool {
	for _, s := range slc {
		if s == e {
			return true
		}
	}
	return false
}
