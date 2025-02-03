package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/akamai/cli/v2/pkg/color"
)

// Colors mapping.
var colorFns = map[slog.Level]func(format string, a ...interface{}) string{
	slog.LevelDebug: color.FaintString,
	slog.LevelInfo:  color.BlueString,
	slog.LevelWarn:  color.YellowString,
	slog.LevelError: color.RedString,
}

var start = time.Now()

// Handler is a custom slog handler
type Handler struct {
	h             slog.Handler
	writer        io.Writer
	goas          []groupOrAttrs
	coloredOutput bool
}
type groupOrAttrs struct {
	group string
	attrs []slog.Attr
}

const timeFormat string = time.RFC3339

// Enabled is a wrapper over slog.Handler.Enabled method
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

// Handle handles the Record
// It will only be called when Enabled returns true
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	colorFn := colorFns[r.Level]
	buf := make([]byte, 0, 1024)

	goas := normalizeGroupsAndAttributes(h.goas, r.NumAttrs())
	for _, goa := range goas {
		if goa.group != "" {
			buf = fmt.Appendf(buf, "%s:\n", goa.group)
		} else {
			for _, a := range goa.attrs {
				buf = h.appendAttr(buf, a, colorFn)
			}
		}
	}

	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a, colorFn)
		return true
	})

	ts := time.Since(start) / time.Second

	if h.coloredOutput {
		_, _ = fmt.Fprintf(h.writer, "%s[%04d] %-25s%s\n", colorFn("%6s", r.Level), ts, r.Message, buf)
	} else {
		t := time.Now().Format(timeFormat)
		_, _ = fmt.Fprintf(h.writer, "[%s] %s %-25s%s\n", t, r.Level, r.Message, buf)
	}

	return nil
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

func (h *Handler) withGroupOrAttrs(goa groupOrAttrs) *Handler {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

func normalizeGroupsAndAttributes(groupOfAttrs []groupOrAttrs, numAttrs int) []groupOrAttrs {
	goas := groupOfAttrs
	if numAttrs == 0 {
		for len(goas) > 0 && goas[len(goas)-1].group != "" {
			goas = goas[:len(goas)-1]
		}
	}
	return goas
}

func (h *Handler) appendAttr(buf []byte, a slog.Attr, color func(format string, a ...interface{}) string) []byte {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf
	}

	if len(buf) != 0 {
		buf = fmt.Append(buf, " ")
	}
	if h.coloredOutput {
		switch a.Value.Kind() {
		case slog.KindTime:
			buf = fmt.Appendf(buf, " %s=%s", color("%s", a.Key), a.Value.Time().Format(timeFormat))
		default:
			buf = fmt.Appendf(buf, " %s=%v", color("%s", a.Key), a.Value)
		}
	} else {
		switch a.Value.Kind() {
		case slog.KindTime:
			buf = fmt.Appendf(buf, " %s=%s", a.Key, a.Value.Time().Format(timeFormat))
		default:
			buf = fmt.Appendf(buf, " %v=%v", a.Key, a.Value)
		}
	}

	return buf

}

func suppressDefaults(
	next func([]string, slog.Attr) slog.Attr,
) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey ||
			a.Key == slog.LevelKey ||
			a.Key == slog.MessageKey {
			return slog.Attr{}
		}
		if next == nil {
			return a
		}
		return next(groups, a)
	}
}

// NewHandler returns configured SlogHandler
func NewHandler(w io.Writer, coloredOutput bool, opts *slog.HandlerOptions) *Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	b := &bytes.Buffer{}

	return &Handler{
		h: slog.NewTextHandler(b, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaults(opts.ReplaceAttr),
		}),
		writer:        w,
		coloredOutput: coloredOutput,
	}
}
