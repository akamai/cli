package log

import (
	"context"
	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"github.com/urfave/cli/v2"
	"os"
	"strings"
)

type Logger log.Interface

func SetupContext(ctx context.Context, app *cli.App) context.Context {
	logger := &log.Logger{
		Level: log.InfoLevel,
	}
	output := app.Writer
	if outputEnv := os.Getenv("AKAMAI_LOG_PATH"); outputEnv != "" {
		f, err := os.OpenFile(outputEnv, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Warn("Invalid value of AKAMAI_LOG_PATH")
		}
		output = f
	}
	if lvlEnv := os.Getenv("AKAMAI_LOG"); lvlEnv != "" {
		logLevel, err := log.ParseLevel(strings.ToLower(lvlEnv))
		if err == nil {
			logger.Level = logLevel
		} else {
			log.Warn("Unknown AKAMAI_LOG value. Allowed values: fatal, error, warn, info, debug")
		}
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
