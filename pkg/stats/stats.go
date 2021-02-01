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
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	time "time"

	"github.com/fatih/color"
	"github.com/google/uuid"

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
)

// Akamai CLI (optionally) tracks upgrades, package installs, and updates anonymously
//
// This is done by generating an anonymous UUID that events are tied to

const (
	statsVersion     string = "1.1"
	sleepTime24Hours        = time.Hour * 24
)

// FirstRunCheckStats ...
func FirstRunCheckStats(ctx context.Context, bannerShown bool) bool {
	term := terminal.Get(ctx)
	anonymous := color.New(color.FgWhite, color.Bold).Sprint("anonymous")

	if config.GetConfigValue("cli", "enable-cli-statistics") == "" {
		if !bannerShown {
			bannerShown = true
			terminal.ShowBanner(ctx)
		}
		term.Printf("Help Akamai improve Akamai CLI by automatically sending %s diagnostics and usage data.\n", anonymous)
		term.Writeln("Examples of data being sent include upgrade statistics, and packages installed and updated.")
		term.Writeln("Note: if you choose to opt-out, a single %s event will be submitted to help track overall usage.\n", anonymous)

		answer, err := term.Confirm(fmt.Sprintf("Send %s diagnostics and usage data to Akamai? [Y/n]: ", anonymous), true)
		if err != nil {
			return bannerShown
		}

		if answer {
			TrackEvent(ctx, "first-run", "stats-opt-out", "true")
			config.SetConfigValue("cli", "enable-cli-statistics", "false")
			if err := config.SaveConfig(ctx); err != nil {
				return false
			}
			return bannerShown
		}

		config.SetConfigValue("cli", "enable-cli-statistics", statsVersion)
		config.SetConfigValue("cli", "stats-version", statsVersion)
		config.SetConfigValue("cli", "last-ping", "never")
		if err := setupUUID(ctx); err != nil {
			return false
		}
		if err := config.SaveConfig(ctx); err != nil {
			return false
		}
		TrackEvent(ctx, "first-run", "stats-enabled", statsVersion)
	} else if config.GetConfigValue("cli", "enable-cli-statistics") != "false" {
		migrateStats(ctx, bannerShown)
	}

	return bannerShown
}

func migrateStats(ctx context.Context, bannerShown bool) bool {
	term := terminal.Get(ctx)

	currentVersion := config.GetConfigValue("cli", "stats-version")
	if currentVersion == statsVersion {
		return bannerShown
	}

	if !bannerShown {
		bannerShown = true
		terminal.ShowBanner(ctx)
	}

	var newStats []string
	if currentVersion == "1.0" {
		newStats = []string{"command name executed (no arguments)", "command version executed"}
	}

	anonymous := color.New(color.FgWhite, color.Bold).Sprint("anonymous")
	term.Printf("Akamai CLI has changed the %s data it collects. It now additionally collects the following: \n\n", anonymous)
	for _, value := range newStats {
		term.Printf(" - %s\n", value)
	}
	term.Printf("\nTo continue collecting %s statistics, Akamai CLI requires that you re-affirm you decision.\n", anonymous)
	term.Writeln("Note: if you choose to opt-out, a single anonymous event will be submitted to help track overall usage.\n")

	answer, err := term.Confirm(fmt.Sprintf("Continue sending %s diagnostics and usage data to Akamai? [Y/n]: ", anonymous), true)
	if err != nil {
		return bannerShown
	}

	if answer {
		TrackEvent(ctx, "first-run", "stats-update-opt-out", statsVersion)
		config.SetConfigValue("cli", "enable-cli-statistics", "false")
		if err := config.SaveConfig(ctx); err != nil {
			return false
		}
		return bannerShown
	}

	config.SetConfigValue("cli", "stats-version", statsVersion)
	if err := config.SaveConfig(ctx); err != nil {
		return false
	}
	TrackEvent(ctx, "first-run", "stats-update-opt-in", statsVersion)

	return bannerShown
}

func setupUUID(ctx context.Context) error {
	if config.GetConfigValue("cli", "client-id") == "" {
		uid, err := uuid.NewRandom()
		if err != nil {
			return err
		}

		config.SetConfigValue("cli", "client-id", uid.String())
		if err := config.SaveConfig(ctx); err != nil {
			return err
		}
	}

	return nil
}

// TrackEvent ...
func TrackEvent(ctx context.Context, category, action, value string) {
	if config.GetConfigValue("cli", "enable-cli-statistics") == "false" {
		return
	}

	term := terminal.Get(ctx)

	clientID := "anonymous"
	if val := config.GetConfigValue("cli", "client-id"); val != "" {
		clientID = val
	}

	form := url.Values{}
	form.Add("tid", "UA-34796267-23")
	form.Add("v", "1")        // Version 1
	form.Add("aip", "1")      // Anonymize IP
	form.Add("cid", clientID) // Client ID
	form.Add("t", "event")    // Type
	form.Add("ec", category)  // Category
	form.Add("ea", action)    // Action
	form.Add("el", value)     // Label

	hc := http.Client{}
	debug := os.Getenv("AKAMAI_CLI_DEBUG_ANALYTICS")
	var req *http.Request
	var err error

	if debug != "" {
		req, err = http.NewRequest(http.MethodPost, "https://www.google-analytics.com/debug/collect", strings.NewReader(form.Encode()))
	} else {
		req, err = http.NewRequest(http.MethodPost, "https://www.google-analytics.com/collect", strings.NewReader(form.Encode()))
	}

	if err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := hc.Do(req)
	if err != nil {
		return
	}
	if debug != "" {
		body, _ := ioutil.ReadAll(res.Body)
		term.Writeln(string(body))
	}
}

// CheckPing ...
func CheckPing(ctx context.Context) error {
	if config.GetConfigValue("cli", "enable-cli-statistics") == "false" {
		return nil
	}

	data := strings.TrimSpace(config.GetConfigValue("cli", "last-ping"))

	doPing := false
	if data == "" || data == "never" {
		doPing = true
	} else {
		configValue := strings.TrimPrefix(strings.TrimSuffix(data, "\""), "\"")
		lastPing, err := time.Parse(time.RFC3339, configValue)
		if err != nil {
			return err
		}

		currentTime := time.Now()
		if lastPing.Add(sleepTime24Hours).Before(currentTime) {
			doPing = true
		}
	}

	if doPing {
		TrackEvent(ctx, "ping", "daily", "pong")
		config.SetConfigValue("cli", "last-ping", time.Now().Format(time.RFC3339))
		if err := config.SaveConfig(ctx); err != nil {
			return err
		}
	}
	return nil
}
