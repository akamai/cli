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

package config

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-ini/ini"

	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
)

const (
	configVersion string = "1.1"
)

type (
	Config interface {
		Save(context.Context) error
		Values() map[string]map[string]string
		GetValue(string, string) (string, bool)
		SetValue(string, string, string)
		UnsetValue(string, string)
		ExportEnv(context.Context) error
	}

	IniConfig struct {
		path string
		file *ini.File
	}

	contextType string
)

var configContext contextType = "config"

func NewIni() (*IniConfig, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}
	if _, err = os.Stat(path); os.IsNotExist(err) {
		iniFile := ini.Empty()
		return &IniConfig{path: path, file: iniFile}, nil
	}
	iniFile, err := ini.Load(path)
	if err != nil {
		return nil, err
	}
	return &IniConfig{path: path, file: iniFile}, nil
}

// Context sets the terminal in the context
func Context(ctx context.Context, cfg Config) context.Context {
	return context.WithValue(ctx, configContext, cfg)
}

// Get gets the terminal from the context
func Get(ctx context.Context) Config {
	t, ok := ctx.Value(configContext).(Config)
	if !ok {
		panic(errors.New("context does not have a Config"))
	}

	return t
}

func (c *IniConfig) Save(ctx context.Context) error {
	term := terminal.Get(ctx)
	if err := c.file.SaveTo(c.path); err != nil {
		term.Writeln(err.Error())
		return err
	}

	return nil
}

func (c *IniConfig) Values() map[string]map[string]string {
	sections := make(map[string]map[string]string)
	for _, section := range c.file.Sections() {
		values := make(map[string]string)
		for _, key := range section.Keys() {
			values[key.Name()] = key.String()
		}
		sections[section.Name()] = values
	}
	return sections
}

func (c *IniConfig) GetValue(section, key string) (string, bool) {
	s := c.file.Section(section)
	if !s.HasKey(key) {
		return "", false
	}
	return s.Key(key).String(), true
}

func (c *IniConfig) SetValue(section, key, value string) {
	s := c.file.Section(section)
	s.Key(key).SetValue(value)
}

func (c *IniConfig) UnsetValue(section, key string) {
	s := c.file.Section(section)
	s.DeleteKey(key)
}

func (c *IniConfig) ExportEnv(ctx context.Context) error {
	if err := migrateConfig(ctx, c); err != nil {
		return err
	}

	for _, section := range c.file.Sections() {
		for _, key := range section.Keys() {
			envVar := "AKAMAI_" + strings.ToUpper(section.Name()) + "_"
			envVar += strings.ToUpper(strings.Replace(key.Name(), "-", "_", -1))
			if err := os.Setenv(envVar, key.String()); err != nil {
				return err
			}
		}
	}
	return nil
}

func getConfigFilePath() (string, error) {
	cliPath, err := tools.GetAkamaiCliPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(cliPath, "config"), nil
}

func migrateConfig(ctx context.Context, cfg *IniConfig) error {
	var currentVersion string
	if _, err := os.Stat(cfg.path); err == nil {
		// Do we need to migrate from an older version?
		currentVersion, _ = cfg.GetValue("cli", "config-version")
		if currentVersion == configVersion {
			return nil
		}
	}

	switch currentVersion {
	case "":
		// Create v1
		cliPath, err := tools.GetAkamaiCliPath()
		if err != nil {
			return err
		}

		var data []byte
		upgradeFile := filepath.Join(cliPath, ".upgrade-check")
		if _, err := os.Stat(upgradeFile); err == nil {
			data, _ = ioutil.ReadFile(upgradeFile)
		} else {
			upgradeFile = filepath.Join(cliPath, ".update-check")
			if _, err := os.Stat(upgradeFile); err == nil {
				data, _ = ioutil.ReadFile(upgradeFile)
			}
		}

		if len(data) != 0 {
			date := string(data)
			if date == "never" || date == "ignore" {
				cfg.SetValue("cli", "last-upgrade-check", date)
			} else {
				if m := strings.LastIndex(date, "m="); m != -1 {
					date = date[0 : m-1]
				}
				lastUpgrade, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", date)
				if err == nil {
					cfg.SetValue("cli", "last-upgrade-check", lastUpgrade.Format(time.RFC3339))
				}
			}

			if err := os.Remove(upgradeFile); err != nil {
				return err
			}
		}

		cfg.SetValue("cli", "config-version", "1")
	case "1":
		// Upgrade to v1.1
		if val, _ := cfg.GetValue("cli", "enable-cli-statistics"); val == "true" {
			cfg.SetValue("cli", "stats-version", "1.0")
		}
		cfg.SetValue("cli", "config-version", "1.1")
	}

	if err := cfg.Save(ctx); err != nil {
		return err
	}
	return migrateConfig(ctx, cfg)
}
