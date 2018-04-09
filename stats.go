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
	"net/http"
	"net/url"
	"strings"
	time "time"

	"github.com/tuvistavie/securerandom"
)

// Akamai CLI (optionally) tracks upgrades, package installs, and updates anonymously
//
// This is done by generating an anonymous UUID that events are tied to

func setupUUID() error {
	if getConfigValue("cli", "client-id") == "" {
		uuid, err := securerandom.Uuid()
		if err != nil {
			return err
		}

		setConfigValue("cli", "client-id", uuid)
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
