package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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
	
	return cliPath + string(os.PathSeparator) + "config", nil
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
		fmt.Println(err.Error())
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

	setConfigValue("config-version", configVersion)
	
	cliPath, _ := getAkamaiCliPath()
	upgradeFile := cliPath + string(os.PathSeparator) + ".upgrade-check"

	if _, err := os.Stat(upgradeFile); err == nil {
		data, err := ioutil.ReadFile(upgradeFile)
		if err == nil {
			setConfigValue("last-upgrade-check", string(data))
		}
	} else {
		setConfigValue("last-upgrade-check", "never")
	}

	saveConfig()
}

func getConfigValue(sectionOrKey string, keyName ...string) string {
	var key string
	if len(keyName) == 0 {
		key = sectionOrKey
		sectionOrKey = "cli"
	} else {
		key = keyName[0]
	}

	config, err := openConfig()
	if err != nil {
		return ""
	}

	section := config.Section(sectionOrKey)
	return section.Key(key).String()
}

func setConfigValue(sectionOrKey string, keyOrValue string, value ...string) {
	var val string
	var key string

	if len(value) == 0 {
		val = keyOrValue
		key = sectionOrKey
		sectionOrKey = "cli"
	} else {
		val = value[0]
		key = keyOrValue
	}

	config, err := openConfig()
	if err != nil {
		return
	}

	section := config.Section(sectionOrKey)
	section.Key(key).SetValue(val)
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