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

var config = make(map[string]*ini.File)

func getConfigFilePath() (string, error) {
	cliPath, err := tools.GetAkamaiCliPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(cliPath, "config"), nil
}

// OpenConfig ..
func OpenConfig() (*ini.File, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	if config[path] != nil {
		return config[path], nil
	}

	if _, err = os.Stat(path); os.IsNotExist(err) {
		iniFile := ini.Empty()
		config[path] = iniFile
		return config[path], nil
	}

	iniFile, err := ini.Load(path)
	if err != nil {
		return nil, err
	}
	config[path] = iniFile

	return config[path], nil
}

// SaveConfig ...
func SaveConfig(ctx context.Context) error {

	term := terminal.Get(ctx)
	config, err := OpenConfig()
	if err != nil {
		return err
	}

	path, err := getConfigFilePath()
	if err != nil {
		return err
	}

	err = config.SaveTo(path)
	if err != nil {
		term.Writeln(err.Error())
		return err
	}

	return nil
}

func migrateConfig(ctx context.Context) error {
	configPath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	_, err = OpenConfig()
	if err != nil {
		return err
	}

	var currentVersion string
	if _, err = os.Stat(configPath); err == nil {
		// Do we need to migrate from an older version?
		currentVersion = GetConfigValue("cli", "config-version")
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
				SetConfigValue("cli", "last-upgrade-check", date)
			} else {
				if m := strings.LastIndex(date, "m="); m != -1 {
					date = date[0 : m-1]
				}
				lastUpgrade, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", date)
				if err == nil {
					SetConfigValue("cli", "last-upgrade-check", lastUpgrade.Format(time.RFC3339))
				}
			}

			if err := os.Remove(upgradeFile); err != nil {
				return err
			}
		}

		SetConfigValue("cli", "config-version", "1")
	case "1":
		// Upgrade to v1.1
		if GetConfigValue("cli", "enable-cli-statistics") == "true" {
			SetConfigValue("cli", "stats-version", "1.0")
		}
		SetConfigValue("cli", "config-version", "1.1")
	}

	if err := SaveConfig(ctx); err != nil {
		return err
	}
	return migrateConfig(ctx)
}

// GetConfigValue ...
func GetConfigValue(sectionName, keyName string) string {
	config, err := OpenConfig()
	if err != nil {
		return ""
	}

	section := config.Section(sectionName)
	key := section.Key(keyName)
	if key != nil {
		return key.String()
	}

	return ""
}

// SetConfigValue ...
func SetConfigValue(sectionName, key, value string) {
	config, err := OpenConfig()
	if err != nil {
		return
	}

	section := config.Section(sectionName)
	section.Key(key).SetValue(value)
}

// UnsetConfigValue ...
func UnsetConfigValue(sectionName, key string) {
	config, err := OpenConfig()
	if err != nil {
		return
	}

	section := config.Section(sectionName)
	section.DeleteKey(key)
}

// ExportConfigEnv ...
func ExportConfigEnv(ctx context.Context) error {
	if err := migrateConfig(ctx); err != nil {
		return err
	}

	config, err := OpenConfig()
	if err != nil {
		return err
	}

	for _, section := range config.Sections() {
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
