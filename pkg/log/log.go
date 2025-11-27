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
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/mattn/go-colorable"
)

// SetupContext creates supplies a context.Context with new Logger instance
// It handles setting up logging level and log output
func SetupContext(ctx context.Context, defaultWriter io.Writer) context.Context {
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelError)
	logger := slog.New(NewHandler(defaultWriter, true, &slog.HandlerOptions{
		Level: lvl,
	}))

	if lvlEnv := os.Getenv("AKAMAI_LOG"); lvlEnv != "" {
		logLevel, err := parseLevel(strings.ToLower(lvlEnv))
		if err == nil {
			lvl.Set(logLevel)
		} else {
			logger.Error(err.Error())
		}
	}
	if outputEnv := os.Getenv("AKAMAI_CLI_LOG_PATH"); outputEnv != "" {
		f, err := os.OpenFile(outputEnv, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			logger.Error(fmt.Sprintf("Invalid value of AKAMAI_CLI_LOG_PATH %s", err))
		}
		logger = slog.New(NewHandler(colorable.NewNonColorable(f), false, &slog.HandlerOptions{
			Level: lvl,
		}))

	}
	return NewContext(ctx, *logger)
}

// WithCommand returns a Logger supplied with given 'command' field
func WithCommand(ctx context.Context, command string) *slog.Logger {
	log := FromContext(ctx)
	return log.With("command", command)
}

// WithCommandContext returns a context withe a logger and supplied with given 'command' field
func WithCommandContext(ctx context.Context, command string) context.Context {
	log := FromContext(ctx)
	return NewContext(ctx, *log.With("command", command))

}

func parseLevel(lvl string) (slog.Level, error) {
	switch lvl {
	case "fatal", "error":
		return slog.LevelError, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	default:
		return slog.LevelError, errors.New("unknown AKAMAI_LOG value. Allowed values: fatal, error, warn, warning, info, debug")
	}

}
