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
	"github.com/akamai/cli/pkg/log"
	"os"

	"github.com/urfave/cli/v2"
)

func cmdHelp(c *cli.Context) error {
	c.Context = log.WithCommandContext(c.Context, c.Command.Name)
	if c.Args().Present() {
		cmd := c.Args().First()

		builtinCmds := getBuiltinCommands(c)
		for _, builtInCmd := range builtinCmds {
			if builtInCmd.Commands[0].Name == cmd {
				return cli.ShowCommandHelp(c, cmd)
			}
		}

		// The arg mangling ensures that aliases are handled
		os.Args = append([]string{os.Args[0], cmd, "help"}, c.Args().Tail()...)
		err := c.App.Run(os.Args)
		return err
	}

	return cli.ShowAppHelp(c)
}
