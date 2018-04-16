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
	"net/http"
	"net/url"
	"strings"
	time "time"

	akamai "github.com/akamai/cli-common-golang"
	"github.com/fatih/color"
	"github.com/google/uuid"
)

// Akamai CLI (optionally) tracks upgrades, package installs, and updates anonymously
//
// This is done by generating an anonymous UUID that events are tied to

func firstRunCheckStats(bannerShown bool) bool {
	if getConfigValue("cli", "client-id") == "" && getConfigValue("cli", "enable-cli-statistics") != "false" {
		if !bannerShown {
			bannerShown = true
			showBanner()
		}
		anonymous := color.New(color.FgWhite, color.Bold).Sprint("anonymous")
		fmt.Fprintf(akamai.App.Writer, "Help Akamai improve Akamai CLI by automatically sending %s diagnostics and usage data.\n", anonymous)
		fmt.Fprintf(akamai.App.Writer, "Examples of data being send include upgrade statistics, and packages installed and updated.\n\n")
		fmt.Fprintf(akamai.App.Writer, "Send %s diagnostics and usage data to Akamai? [Y/n]: ", anonymous)

		answer := ""
		fmt.Scanln(&answer)
		if answer != "" && strings.ToLower(answer) != "y" {
			setConfigValue("cli", "enable-cli-statistics", "false")
			saveConfig()
			return bannerShown
		}

		setConfigValue("cli", "enable-cli-statistics", "true")
		setConfigValue("cli", "last-ping", "never")
		setupUUID()
		saveConfig()
		trackEvent("first-run", "true")
	}

	return bannerShown
}

func setupUUID() error {
	if getConfigValue("cli", "client-id") == "" {
		uuid, err := uuid.NewRandom()
		if err != nil {
			return err
		}

		setConfigValue("cli", "client-id", uuid.String())
		saveConfig()
	}

	return nil
}

func trackEvent(action string, value string) {
	if getConfigValue("cli", "enable-cli-statistics") == "false" {
		return
	}

	form := url.Values{}
	form.Add("tid", "UA-34796267-23")
	form.Add("v", "1")                                  // Version 1
	form.Add("aip", "1")                                // Anonymize IP
	form.Add("cid", getConfigValue("cli", "client-id")) // Unique Cilent ID
	form.Add("t", "event")                              // Type
	form.Add("ec", "akamai-cli")                        // Category
	form.Add("ea", action)                              // Action
	form.Add("el", value)                               // Label

	hc := http.Client{}
	req, err := http.NewRequest("POST", "https://www.google-analytics.com/collect", strings.NewReader(form.Encode()))
	if err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	hc.Do(req)
}

func checkPing() {
	if getConfigValue("cli", "enable-cli-statistics") == "false" {
		return
	}

	data := strings.TrimSpace(getConfigValue("cli", "last-ping"))

	doPing := false
	if data == "" || data == "never" {
		doPing = true
	}

	if !doPing {
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
		trackEvent("ping", "pong")
		setConfigValue("cli", "last-ping", time.Now().Format(time.RFC3339))
		saveConfig()
	}
}
