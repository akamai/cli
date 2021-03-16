package commands

import (
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"os"
	"testing"
)

func TestCmdSubcommand(t *testing.T) {
	tests := map[string]struct {
		command   string
		args      []string
		init      func(*testing.T, *mocked)
		withError string
	}{
		"run installed akamai echo command as binary": {
			command: "echo",
			args:    []string{"abc"},
			init: func(t *testing.T, m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
			},
		},
		"run installed akamai echo command as binary with alias": {
			command: "e",
			args:    []string{"abc"},
			init: func(t *testing.T, m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
			},
		},
		"run installed akamai echo command with python required": {
			command: "echo-python",
			args:    []string{"abc"},
			init: func(t *testing.T, m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
			},
		},
		"run installed akamai echo command as .cmd file": {
			command: "echo-cmd",
			args:    []string{"abc"},
			init: func(t *testing.T, m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, "testdata/.akamai-cli/src/cli-echo/bin/akamai-echo-cmd.cmd").
					Return([]string{"testdata/.akamai-cli/src/cli-echo/bin/akamai-echo-cmd.cmd"}, nil)
			},
		},
		"run installed python akamai echo command": {
			command: "echo-cmd",
			args:    []string{"abc"},
			init: func(t *testing.T, m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, "testdata/.akamai-cli/src/cli-echo/bin/akamai-echo-cmd.cmd").
					Return([]string{"testdata/.akamai-cli/src/cli-echo/bin/akamai-echo-cmd.cmd"}, nil)
			},
		},
		"executable not found": {
			command:   "invalid",
			args:      []string{"abc"},
			init:      func(t *testing.T, m *mocked) {},
			withError: `Executable "invalid" not found`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
			m := &mocked{&terminal.Mock{}, &config.Mock{}, &git.Mock{}, &packages.Mock{}}
			command := &cli.Command{
				Name:   test.command,
				Action: cmdSubcommand(m.gitRepo, m.langManager),
			}
			app, ctx := setupTestApp(command, m)
			args := os.Args[0:1]
			args = append(args, test.command)
			args = append(args, test.args...)

			test.init(t, m)
			err := app.RunContext(ctx, args)

			m.cfg.AssertExpectations(t)
			if test.withError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.withError)
				return
			}
			require.NoError(t, err)
		})
	}
}
