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
	"fmt"
	"strings"
	"time"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/config"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/urfave/cli/v2"
)

func cmdConfigSet(c *cli.Context) (e error) {
	c.Context = log.WithCommandContext(c.Context, c.Command.Name)
	logger := log.FromContext(c.Context)
	start := time.Now()
	logger.Debug("CONFIG SET START")
	defer func() {
		if e == nil {
			logger.Debug(fmt.Sprintf("CONFIG SET FINISH: %v", time.Since(start)))
		} else {
			logger.Error(fmt.Sprintf("CONFIG SET ERROR: %v", e))
		}
	}()
	cfg := config.Get(c.Context)

	section, key, err := parseConfigPath(c)
	if err != nil {
		logger.Error(fmt.Sprintf("Error parsing config path: %v", err))
		return cli.Exit(color.RedString("Unable to set config value: %v", err), 1)
	}

	value := strings.Join(c.Args().Tail(), " ")
	cfg.SetValue(section, key, value)
	if err := cfg.Save(c.Context); err != nil {
		logger.Error(fmt.Sprintf("Error saving config: %v", err))
		return cli.Exit(color.RedString("Unable to set config value: %v", err), 1)
	}

	return nil
}

func cmdConfigGet(c *cli.Context) (e error) {
	c.Context = log.WithCommandContext(c.Context, c.Command.Name)
	logger := log.FromContext(c.Context)
	start := time.Now()
	logger.Debug("CONFIG GET START")
	defer func() {
		if e == nil {
			logger.Debug(fmt.Sprintf("CONFIG GET FINISH: %v", time.Since(start)))
		} else {
			logger.Error(fmt.Sprintf("CONFIG GET ERROR: %v", e))
		}
	}()
	cfg := config.Get(c.Context)

	section, key, err := parseConfigPath(c)
	if err != nil {
		logger.Error(fmt.Sprintf("Error parsing config path: %v", err))
		return cli.Exit(color.RedString("Unable to get config value: %v", err), 1)
	}

	val, _ := cfg.GetValue(section, key)
	if _, err := terminal.Get(c.Context).Writeln(val); err != nil {
		return err
	}
	logger.Debug(val)

	return nil
}

func cmdConfigUnset(c *cli.Context) (e error) {
	c.Context = log.WithCommandContext(c.Context, c.Command.Name)
	logger := log.FromContext(c.Context)
	start := time.Now()
	logger.Debug("CONFIG UNSET START")
	defer func() {
		if e == nil {
			logger.Debug(fmt.Sprintf("CONFIG UNSET FINISH: %v", time.Since(start)))
		} else {
			logger.Error(fmt.Sprintf("CONFIG UNSET ERROR: %v", e))
		}
	}()
	cfg := config.Get(c.Context)
	section, key, err := parseConfigPath(c)
	if err != nil {
		logger.Error(fmt.Sprintf("Error parsing config path: %v", err))
		return cli.Exit(color.RedString("Unable to unset config value: %v", err), 1)
	}

	cfg.UnsetValue(section, key)
	if err := cfg.Save(c.Context); err != nil {
		logger.Error(fmt.Sprintf("Error saving config: %v", err))
		return cli.Exit(color.RedString("Unable to set config value: %v", err), 1)
	}
	return nil
}

func cmdConfigList(c *cli.Context) (e error) {
	c.Context = log.WithCommandContext(c.Context, c.Command.Name)
	logger := log.FromContext(c.Context)
	start := time.Now()
	logger.Debug("CONFIG LIST START")
	defer func() {
		if e == nil {
			logger.Debug(fmt.Sprintf("CONFIG LIST FINISH: %v", time.Since(start)))
		} else {
			logger.Error(fmt.Sprintf("CONFIG LIST ERROR: %v", e))
		}
	}()
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

func parseConfigPath(c *cli.Context) (string, string, error) {
	path := strings.Split(c.Args().First(), ".")
	if len(path) < 2 {
		return "", "", fmt.Errorf("section key has to be provided in <section>.<key> format")
	}
	section := path[0]
	key := strings.Join(path[1:], "-")
	return section, key, nil
}
