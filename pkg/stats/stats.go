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

package stats

import (
	"fmt"
	"github.com/akamai/cli/pkg/app"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	time "time"

	"github.com/fatih/color"
	"github.com/google/uuid"
)

// Akamai CLI (optionally) tracks upgrades, package installs, and updates anonymously
//
// This is done by generating an anonymous UUID that events are tied to

const statsVersion string = "1.1"

func FirstRunCheckStats(bannerShown bool) bool {
	anonymous := color.New(color.FgWhite, color.Bold).Sprint("anonymous")

	if config.GetConfigValue("cli", "enable-cli-statistics") == "" {
		if !bannerShown {
			bannerShown = true
			io.ShowBanner()
		}
		fmt.Fprintf(app.App.Writer, "Help Akamai improve Akamai CLI by automatically sending %s diagnostics and usage data.\n", anonymous)
		fmt.Fprintln(app.App.Writer, "Examples of data being sent include upgrade statistics, and packages installed and updated.")
		fmt.Fprintf(app.App.Writer, "Note: if you choose to opt-out, a single %s event will be submitted to help track overall usage.", anonymous)
		fmt.Fprintf(app.App.Writer, "\n\nSend %s diagnostics and usage data to Akamai? [Y/n]: ", anonymous)

		answer := ""
		fmt.Scanln(&answer)
		if answer != "" && strings.ToLower(answer) != "y" {
			TrackEvent("first-run", "stats-opt-out", "true")
			config.SetConfigValue("cli", "enable-cli-statistics", "false")
			config.SaveConfig()
			return bannerShown
		}

		config.SetConfigValue("cli", "enable-cli-statistics", statsVersion)
		config.SetConfigValue("cli", "stats-version", statsVersion)
		config.SetConfigValue("cli", "last-ping", "never")
		setupUUID()
		config.SaveConfig()
		TrackEvent("first-run", "stats-enabled", statsVersion)
	} else if config.GetConfigValue("cli", "enable-cli-statistics") != "false" {
		migrateStats(bannerShown)
	}

	return bannerShown
}

func migrateStats(bannerShown bool) bool {
	currentVersion := config.GetConfigValue("cli", "stats-version")
	if currentVersion == statsVersion {
		return bannerShown
	}

	if !bannerShown {
		bannerShown = true
		io.ShowBanner()
	}

	var newStats []string
	switch currentVersion {
	case "1.0":
		newStats = []string{"command name executed (no arguments)", "command version executed"}
	}

	anonymous := color.New(color.FgWhite, color.Bold).Sprint("anonymous")
	fmt.Fprintf(app.App.Writer, "Akamai CLI has changed the %s data it collects. It now additionally collects the following: \n\n", anonymous)
	for _, value := range newStats {
		fmt.Fprintf(app.App.Writer, " - %s\n", value)
	}
	fmt.Fprintf(app.App.Writer, "\nTo continue collecting %s statistics, Akamai CLI requires that you re-affirm you decision.\n", anonymous)
	fmt.Fprintln(app.App.Writer, "Note: if you choose to opt-out, a single anonymous event will be submitted to help track overall usage.")
	fmt.Fprintf(app.App.Writer, "\nContinue sending %s diagnostics and usage data to Akamai? [Y/n]: ", anonymous)

	answer := ""
	fmt.Scanln(&answer)
	if answer != "" && strings.ToLower(answer) != "y" {
		TrackEvent("first-run", "stats-update-opt-out", statsVersion)
		config.SetConfigValue("cli", "enable-cli-statistics", "false")
		config.SaveConfig()
		return bannerShown
	}

	config.SetConfigValue("cli", "stats-version", statsVersion)
	config.SaveConfig()
	TrackEvent("first-run", "stats-update-opt-in", statsVersion)

	return bannerShown
}

func setupUUID() error {
	if config.GetConfigValue("cli", "client-id") == "" {
		uuid, err := uuid.NewRandom()
		if err != nil {
			return err
		}

		config.SetConfigValue("cli", "client-id", uuid.String())
		config.SaveConfig()
	}

	return nil
}

func TrackEvent(category string, action string, value string) {
	if config.GetConfigValue("cli", "enable-cli-statistics") == "false" {
		return
	}

	clientId := "anonymous"
	if val := config.GetConfigValue("cli", "client-id"); val != "" {
		clientId = val
	}

	form := url.Values{}
	form.Add("tid", "UA-34796267-23")
	form.Add("v", "1")        // Version 1
	form.Add("aip", "1")      // Anonymize IP
	form.Add("cid", clientId) // Client ID
	form.Add("t", "event")    // Type
	form.Add("ec", category)  // Category
	form.Add("ea", action)    // Action
	form.Add("el", value)     // Label

	hc := http.Client{}
	debug := os.Getenv("AKAMAI_CLI_DEBUG_ANALYTICS")
	var req *http.Request
	var err error

	if debug != "" {
		req, err = http.NewRequest("POST", "https://www.google-analytics.com/debug/collect", strings.NewReader(form.Encode()))
	} else {
		req, err = http.NewRequest("POST", "https://www.google-analytics.com/collect", strings.NewReader(form.Encode()))
	}

	if err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := hc.Do(req)
	if debug != "" {
		body, _ := ioutil.ReadAll(res.Body)
		fmt.Fprintln(app.App.Writer, string(body))
	}
}

func CheckPing() {
	if config.GetConfigValue("cli", "enable-cli-statistics") == "false" {
		return
	}

	data := strings.TrimSpace(config.GetConfigValue("cli", "last-ping"))

	doPing := false
	if data == "" || data == "never" {
		doPing = true
	} else {
		configValue := strings.TrimPrefix(strings.TrimSuffix(string(data), "\""), "\"")
		lastPing, err := time.Parse(time.RFC3339, configValue)
		if err != nil {
			return
		}

		currentTime := time.Now()
		if lastPing.Add(time.Hour * 24).Before(currentTime) {
			doPing = true
		}
	}

	if doPing {
		TrackEvent("ping", "daily", "pong")
		config.SetConfigValue("cli", "last-ping", time.Now().Format(time.RFC3339))
		config.SaveConfig()
	}
}
