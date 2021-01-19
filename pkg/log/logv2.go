package log

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"github.com/urfave/cli/v2"
	"os"
)

func SetupContext(ctx context.Context, app *cli.App) context.Context {
	logger := &log.Logger{
		Level: log.InfoLevel,
	}
	if lvlEnv := os.Getenv("AKAMAI_LOG"); lvlEnv != "" {
		logLevel, err := log.ParseLevel(lvlEnv)
		if err == nil {
			logger.Level = logLevel
		} else {
			fmt.Fprintln(app.Writer, "[WARN] Unknown AKAMAI_LOG value. Allowed values: panic, fatal, error, warn, info, debug, trace")
		}
	}
	logger.Handler = text.New(app.Writer)
	return log.NewContext(ctx, logger)
}
