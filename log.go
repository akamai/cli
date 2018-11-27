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
	"bufio"
	"fmt"
	"os"
	"strings"

	akamai "github.com/akamai/cli-common-golang"
	log "github.com/sirupsen/logrus"
)

var logBuffer *bufio.Writer

func setupLogging() {
	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation:    true,
		EnvironmentOverrideColors: true,
	})

	log.SetOutput(akamai.App.ErrWriter)

	log.SetLevel(log.PanicLevel)
	if logLevel := os.Getenv("AKAMAI_LOG"); logLevel != "" {
		level, err := log.ParseLevel(logLevel)
		if err == nil {
			log.SetLevel(level)
		} else {
			fmt.Fprintln(akamai.App.Writer, "[WARN] Unknown AKAMAI_LOG value. Allowed values: panic, fatal, error, warn, info, debug, trace")
		}
	}
}

func logMultiline(f func(args ...interface{}), args ...string) {
	for _, str := range args {
		for _, str := range strings.Split(strings.Trim(str, "\n"), "\n") {
			f(str)
		}
	}
}

func logMultilineln(f func(args ...interface{}), args ...string) {
	logMultiline(f, args...)
}

func logMultilinef(f func(formatter string, args ...interface{}), formatter string, args ...interface{}) {
	str := fmt.Sprintf(formatter, args...)
	for _, str := range strings.Split(strings.Trim(str, "\n"), "\n") {
		f(str)
	}
}
