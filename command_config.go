package main

import (
	"fmt"
	"strings"

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

	fmt.Println(getConfigValue(section, key))
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
			fmt.Printf("%s.%s = %s\n", sectionName, key.Name(), key.Value())
		}

		return
	}

	for _, section := range config.Sections() {
		for _, key := range section.Keys() {
			fmt.Printf("%s.%s = %s\n", section.Name(), key.Name(), key.Value())
		}
	}
}

func parseConfigPath(c *cli.Context) (string, string) {
	path := strings.Split(c.Args().First(), ".")
	section := path[0]
	key := strings.Join(path[1:], "-")
	return section, key
}
