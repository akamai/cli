package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-ini/ini"
)

const (
	configVersion string = "1"
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

	if _, err := os.Stat(path); os.IsNotExist(err) {
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
		fmt.Fprintln(app.Writer, err.Error())
		return err
	}

	return nil
}

func migrateConfig() {
	configPath, err := getConfigFilePath()
	if err != nil {
		return
	}

	if _, err := os.Stat(configPath); err == nil {
		// Do we need to migrate from an older version?
		if getConfigValue("cli", "config-version") == configVersion {
			return
		}
	}

	// Create v1
	_, err = openConfig()
	if err != nil {
		return
	}

	setConfigValue("cli", "config-version", configVersion)
	saveConfig()

	cliPath, _ := getAkamaiCliPath()

	var data []byte
	upgradeFile := filepath.Join(cliPath, ".upgrade-check")
	if _, err := os.Stat(upgradeFile); err == nil {
		data, err = ioutil.ReadFile(upgradeFile)
	} else {
		upgradeFile = filepath.Join(cliPath, ".update-check")
		if _, err := os.Stat(upgradeFile); err == nil {
			data, err = ioutil.ReadFile(upgradeFile)
		} else {
			return
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

	saveConfig()
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

func addConfigComment(sectionName string, key string, comment string) {
	config, err := openConfig()
	if err != nil {
		return
	}

	section := config.Section(sectionName)
	configKey, err := section.GetKey(key)
	if err != nil {
		return
	}
	configKey.Comment = comment
}

func exportConfigEnv() {
	migrateConfig()

	config, err := openConfig()
	if err != nil {
		return
	}

	for _, section := range config.Sections() {
		envVar := "AKAMAI_" + strings.ToUpper(section.Name()) + "_"

		for _, key := range section.Keys() {
			envVar += strings.ToUpper(strings.Replace(key.Name(), "-", "_", -1))
			os.Setenv(envVar, key.String())
		}
	}
}
