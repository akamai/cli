package log

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const logPath = "./testlogs.txt"

func TestSetupContext(t *testing.T) {
	tests := map[string]struct {
		envs          map[string]string
		expectedLevel slog.Level
		withError     *regexp.Regexp
	}{
		"no envs passed, defaults are used": {
			expectedLevel: slog.LevelError,
		},
		"debug level set": {
			envs:          map[string]string{"AKAMAI_LOG": "DEBUG"},
			expectedLevel: slog.LevelDebug,
		},
		"debug level set, write logs to a file": {
			envs:          map[string]string{"AKAMAI_LOG": "DEBUG", "AKAMAI_CLI_LOG_PATH": logPath},
			expectedLevel: slog.LevelDebug,
		},
		"invalid path passed": {
			envs:          map[string]string{"AKAMAI_CLI_LOG_PATH": ".", "AKAMAI_LOG": "INFO"},
			expectedLevel: slog.LevelInfo,
			withError:     regexp.MustCompile(`ERROR.*Invalid value of AKAMAI_CLI_LOG_PATH`),
		},
		"invalid log level passed, output to terminal": {
			envs:          map[string]string{"AKAMAI_LOG": "abc"},
			expectedLevel: slog.LevelError,
			withError:     regexp.MustCompile(`ERROR.*unknown AKAMAI_LOG value. Allowed values: fatal, error, warn, warning, info, debug`),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for k, v := range test.envs {
				require.NoError(t, os.Setenv(k, v))
			}
			defer func() {
				for k := range test.envs {
					require.NoError(t, os.Unsetenv(k))
				}
			}()
			var buf bytes.Buffer
			ctx := SetupContext(context.Background(), &buf)
			logger := FromContext(ctx)
			assert.True(t, logger.Enabled(ctx, test.expectedLevel))
			if test.withError != nil {
				assert.Regexp(t, test.withError, buf.String())
				return
			}
			logger.Error("test!")
			if v, ok := test.envs["AKAMAI_CLI_LOG_PATH"]; ok {
				res, err := os.ReadFile(v)
				require.NoError(t, err)
				assert.Contains(t, string(res), "test!")
				return
			}
			assert.Contains(t, buf.String(), "test!")
		})
	}
}

func TestWithCommand(t *testing.T) {
	tests := map[string]struct {
		logFile  string
		expected *regexp.Regexp
	}{
		"output to terminal": {
			expected: regexp.MustCompile(` ERROR\[\d{4}] abc *command=test`),
		},
		"output to file": {
			logFile:  logPath,
			expected: regexp.MustCompile(`\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*Z*] ERROR abc *command=test`),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, os.Setenv("AKAMAI_CLI_LOG_PATH", test.logFile))
			defer func() {
				require.NoError(t, os.Unsetenv("AKAMAI_CLI_LOG_PATH"))
			}()
			var buf bytes.Buffer
			ctx := SetupContext(context.Background(), &buf)
			logger := WithCommand(ctx, "test")
			logger.Error("abc")
			if test.logFile != "" {
				res, err := os.ReadFile(test.logFile)
				require.NoError(t, err)
				assert.Regexp(t, test.expected, string(res))
				return
			}
			assert.Regexp(t, test.expected, buf.String())
		})
	}
}
