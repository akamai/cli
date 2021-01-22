package log

import (
	"context"
	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"io"
	"os"
	"strings"
)

type Logger log.Interface

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
	if outputEnv := os.Getenv("AKAMAI_LOG_PATH"); outputEnv != "" {
		f, err := os.OpenFile(outputEnv, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			logger.Warnf("Invalid value of AKAMAI_LOG_PATH %s", err)
		}
		output = f
	}
	logger.Handler = text.New(output)
	return log.NewContext(ctx, logger)
}

func FromContext(ctx context.Context) Logger {
	return log.FromContext(ctx)
}

func WithCommand(ctx context.Context, command string) Logger {
	logger := log.FromContext(ctx)
	return logger.WithField("command", command)
}
