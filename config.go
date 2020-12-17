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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	akamai "github.com/akamai/cli-common-golang"
	"github.com/go-ini/ini"
)

const (
	configVersion string = "1.1"
)

var config map[string]*ini.File = make(map[string]*ini.File)

func getConfigFilePath() (string, error) {
	cliPath, err := getAkamaiCliPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(cliPath, "config"), nil
}

func openConfig() (*ini.File, error) {
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

func saveConfig() error {
	config, err := openConfig()
	if err != nil {
		return err
	}

	path, err := getConfigFilePath()
	if err != nil {
		return err
	}

	err = config.SaveTo(path)
	if err != nil {
		fmt.Fprintln(akamai.App.Writer, err.Error())
		return err
	}

	return nil
}

func migrateConfig() {
	configPath, err := getConfigFilePath()
	if err != nil {
		return
	}

	_, err = openConfig()
	if err != nil {
		return
	}

	var currentVersion string
	if _, err = os.Stat(configPath); err == nil {
		// Do we need to migrate from an older version?
		currentVersion = getConfigValue("cli", "config-version")
		if currentVersion == configVersion {
			return
		}
	}

	switch currentVersion {
	case "":
		// Create v1
		cliPath, _ := getAkamaiCliPath()

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
				setConfigValue("cli", "last-upgrade-check", date)
			} else {
				if m := strings.LastIndex(date, "m="); m != -1 {
					date = date[0 : m-1]
				}
				lastUpgrade, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", date)
				if err == nil {
					setConfigValue("cli", "last-upgrade-check", lastUpgrade.Format(time.RFC3339))
				}
			}

			os.Remove(upgradeFile)
		}

		setConfigValue("cli", "config-version", "1")
	case "1":
		// Upgrade to v1.1
		if getConfigValue("cli", "enable-cli-statistics") == "true" {
			setConfigValue("cli", "stats-version", "1.0")
		}
		setConfigValue("cli", "config-version", "1.1")
	}

	saveConfig()
	migrateConfig()
}

func getConfigValue(sectionName string, keyName string) string {
	config, err := openConfig()
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

func setConfigValue(sectionName string, key string, value string) {
	config, err := openConfig()
	if err != nil {
		return
	}

	section := config.Section(sectionName)
	section.Key(key).SetValue(value)
}

func unsetConfigValue(sectionName string, key string) {
	config, err := openConfig()
	if err != nil {
		return
	}

	section := config.Section(sectionName)
	section.DeleteKey(key)
}

func exportConfigEnv() {
	migrateConfig()

	config, err := openConfig()
	if err != nil {
		return
	}

	for _, section := range config.Sections() {
		for _, key := range section.Keys() {
			envVar := "AKAMAI_" + strings.ToUpper(section.Name()) + "_"
			envVar += strings.ToUpper(strings.Replace(key.Name(), "-", "_", -1))
			os.Setenv(envVar, key.String())
		}
	}
}
