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
	"strings"

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"

	"github.com/urfave/cli/v2"
)

func cmdConfigSet(c *cli.Context) error {
	cfg := config.Get(c.Context)
	section, key := parseConfigPath(c)
	value := strings.Join(c.Args().Tail(), " ")
	cfg.SetValue(section, key, value)
	return cfg.Save(c.Context)
}

func cmdConfigGet(c *cli.Context) error {
	cfg := config.Get(c.Context)
	section, key := parseConfigPath(c)
	val, _ := cfg.GetValue(section, key)
	terminal.Get(c.Context).Writeln(val)
	return nil
}

func cmdConfigUnset(c *cli.Context) error {
	cfg := config.Get(c.Context)
	section, key := parseConfigPath(c)

	cfg.UnsetValue(section, key)
	return cfg.Save(c.Context)
}

func cmdConfigList(c *cli.Context) error {
	cfg := config.Get(c.Context)
	term := terminal.Get(c.Context)

	allValues := cfg.Values()
	if c.NArg() > 0 {
		sectionName := c.Args().First()
		section, ok := allValues[sectionName]
		if !ok {
			return nil
		}
		for key, value := range section {
			term.Printf("%s.%s = %s\n", sectionName, key, value)
		}

		return nil
	}

	for sectionName, section := range allValues {
		for key, value := range section {
			term.Printf("%s.%s = %s\n", sectionName, key, value)
		}
	}
	return nil
}

func parseConfigPath(c *cli.Context) (section, key string) {
	path := strings.Split(c.Args().First(), ".")
	section = path[0]
	key = strings.Join(path[1:], "-")
	return section, key
}
