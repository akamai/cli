package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/tuvistavie/securerandom"
)

// Akamai CLI (optionally) tracks upgrades, package installs, and updates anonymously
//
// This is done by generating an anonymous UUID that events are tied to

func setupUuid() error {
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
	form.Add("tid", "UA-34796267-20")
	form.Add("v", "1") // Version 1
	form.Add("aip", "1") // Anonymize IP
	form.Add("cid", getConfigValue("cli", "client-id")) // Unique Cilent ID
	form.Add("t", "event") // Type
	form.Add("ec", "akamai-cli") // Category
	form.Add("ea", action) // Action
	form.Add("el", value) // Label

	hc := http.Client{}
	req, err := http.NewRequest("POST", "https://www.google-analytics.com/collect", strings.NewReader(form.Encode()))
	if err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	hc.Do(req)
}