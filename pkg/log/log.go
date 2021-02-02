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
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
)

var start = time.Now()

type (
	// Logger is a type wrapper around log.Interface used to simplify imports in the project
	Logger log.Interface

	// Handler is a custom handler designed to work with or without colored output
	Handler struct {
		mu         sync.Mutex
		Writer     io.Writer
		withColors bool
	}
)

// SetupContext creates supplies a context.Context with new Logger instance
// It handles setting up logging level and log output
func SetupContext(ctx context.Context, defaultWriter io.Writer) context.Context {
	logger := &log.Logger{
		Level:   log.InfoLevel,
		Handler: text.New(defaultWriter),
	}
	output := defaultWriter
	if lvlEnv := os.Getenv("AKAMAI_LOG"); lvlEnv != "" {
		logLevel, err := log.ParseLevel(strings.ToLower(lvlEnv))
		if err == nil {
			logger.Level = logLevel
		} else {
			logger.Warn("Unknown AKAMAI_LOG value. Allowed values: fatal, error, warn, info, debug")
		}
	}
	coloredOutput := true
	if outputEnv := os.Getenv("AKAMAI_LOG_PATH"); outputEnv != "" {
		f, err := os.OpenFile(outputEnv, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			logger.Warnf("Invalid value of AKAMAI_LOG_PATH %s", err)
		}
		coloredOutput = false
		output = f
	}
	logger.Handler = NewHandler(output, coloredOutput)
	return log.NewContext(ctx, logger)
}

// FromContext wraps log.FromContext function to simplify imports in the project
func FromContext(ctx context.Context) Logger {
	return log.FromContext(ctx)
}

// WithCommand returns a Logger supplied with given 'command' field
func WithCommand(ctx context.Context, command string) Logger {
	logger := log.FromContext(ctx)
	return logger.WithField("command", command)
}

// WithCommandContext returns a context withe a logger and supplied with given 'command' field
func WithCommandContext(ctx context.Context, command string) context.Context {
	logger := log.FromContext(ctx)
	return log.NewContext(ctx, logger.WithField("command", command))
}

// NewHandler creates a new Handler instance with given parameters
func NewHandler(w io.Writer, withColors bool) *Handler {
	return &Handler{
		Writer:     w,
		withColors: withColors,
	}
}

// HandleLog works the same way as text.Handler from apex/log, but additionally disables coloring output when writing to a text file
func (h *Handler) HandleLog(e *log.Entry) error {
	color := text.Colors[e.Level]
	level := text.Strings[e.Level]
	names := e.Fields.Names()

	h.mu.Lock()
	defer h.mu.Unlock()

	ts := time.Since(start) / time.Second

	if h.withColors {
		fmt.Fprintf(h.Writer, "\033[%dm%6s\033[0m[%04d] %-25s", color, level, ts, e.Message)
	} else {
		t := time.Now().Format(time.RFC3339)
		fmt.Fprintf(h.Writer, "[%s] %s %-25s", t, level, e.Message)
	}

	for _, name := range names {
		if h.withColors {
			fmt.Fprintf(h.Writer, " \033[%dm%s\033[0m=%v", color, name, e.Fields.Get(name))
		} else {
			fmt.Fprintf(h.Writer, " %s=%v", name, e.Fields.Get(name))
		}
	}

	fmt.Fprintln(h.Writer)

	return nil
}
