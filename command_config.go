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
	"fmt"
	"strings"

	akamai "github.com/akamai/cli-common-golang"
	"github.com/urfave/cli"
)

func cmdConfigSet(c *cli.Context) {
	section, key := parseConfigPath(c)

	value := strings.Join(c.Args().Tail(), " ")

	setConfigValue(section, key, value)
	saveConfig()
}

func cmdConfigGet(c *cli.Context) {
	section, key := parseConfigPath(c)

	fmt.Fprintln(akamai.App.Writer, getConfigValue(section, key))
}

func cmdConfigUnset(c *cli.Context) {
	section, key := parseConfigPath(c)

	unsetConfigValue(section, key)
	saveConfig()
}

func cmdConfigList(c *cli.Context) {
	config, err := openConfig()
	if err != nil {
		return
	}

	if c.NArg() > 0 {
		sectionName := c.Args().First()
		section := config.Section(sectionName)
		for _, key := range section.Keys() {
			fmt.Fprintf(akamai.App.Writer, "%s.%s = %s\n", sectionName, key.Name(), key.Value())
		}

		return
	}

	for _, section := range config.Sections() {
		for _, key := range section.Keys() {
			fmt.Fprintf(akamai.App.Writer, "%s.%s = %s\n", section.Name(), key.Name(), key.Value())
		}
	}
}

func parseConfigPath(c *cli.Context) (string, string) {
	path := strings.Split(c.Args().First(), ".")
	section := path[0]
	key := strings.Join(path[1:], "-")
	return section, key
}
