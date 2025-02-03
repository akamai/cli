package log

import (
	"context"
	"log/slog"
	"os"
)

// logKey is a private context key.
type logKey struct{}

// NewContext returns a new context with logger.
func NewContext(ctx context.Context, v slog.Logger) context.Context {
	return context.WithValue(ctx, logKey{}, &v)
}

// FromContext returns the logger from context, or new logger if not present in context
func FromContext(ctx context.Context) *slog.Logger {
	if v, ok := ctx.Value(logKey{}).(*slog.Logger); ok {
		return v
	}
	return slog.New(NewHandler(os.Stdout, false, nil))
}
