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

package log

import (
	"bufio"
	"fmt"
	"github.com/akamai/cli/pkg/app"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

var logBuffer *bufio.Writer

func Setup() {
	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation:    true,
		EnvironmentOverrideColors: true,
	})

	log.SetOutput(app.App.Writer)

	log.SetLevel(log.PanicLevel)
	if logLevel := os.Getenv("AKAMAI_LOG"); logLevel != "" {
		level, err := log.ParseLevel(logLevel)
		if err == nil {
			log.SetLevel(level)
		} else {
			fmt.Fprintln(app.App.Writer, "[WARN] Unknown AKAMAI_LOG value. Allowed values: panic, fatal, error, warn, info, debug, trace")
		}
	}
}

func LogMultiline(f func(args ...interface{}), args ...string) {
	for _, str := range args {
		for _, str := range strings.Split(strings.Trim(str, "\n"), "\n") {
			f(str)
		}
	}
}

func LogMultilineln(f func(args ...interface{}), args ...string) {
	LogMultiline(f, args...)
}

func LogMultilinef(f func(formatter string, args ...interface{}), formatter string, args ...interface{}) {
	str := fmt.Sprintf(formatter, args...)
	for _, str := range strings.Split(strings.Trim(str, "\n"), "\n") {
		f(str)
	}
}