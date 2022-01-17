package commands

import (
	"flag"
	"os"
	"testing"

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCmdSubcommand(t *testing.T) {
	tests := map[string]struct {
		command        string
		args           []string
		init           func(*testing.T, *mocked)
		edgercLocation string
		section        string
		withError      string
	}{
		"run installed akamai echo command as binary": {
			command: "echo",
			args:    []string{"abc"},
			init: func(t *testing.T, m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
			},
		},
		"run installed akamai echo command as binary with edgerc location": {
			command: "echo",
			args:    []string{"abc"},
			init: func(t *testing.T, m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
			},
		},
		"run installed akamai echo command as binary with alias": {
			command:        "e",
			args:           []string{"abc"},
			edgercLocation: "some/location",
			section:        "some_section",
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
			if test.edgercLocation != "" {
				args = append(args, "--edgerc", test.edgercLocation)
			}
			if test.section != "" {
				args = append(args, "--section", test.section)
			}
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

func TestPrepareCommand(t *testing.T) {
	tests := map[string]struct {
		flagsInCtx   map[string]string
		givenCommand []string
		givenArgs    []string
		givenFlags   []string
		expected     []string
	}{
		"no flags and no args": {
			givenCommand: []string{"some", "command"},
			expected:     []string{"some", "command"},
		},
		"no flags and with args": {
			givenCommand: []string{"some", "command"},
			givenArgs:    []string{"--flag_1", "existing_value"},
			expected:     []string{"some", "command", "--flag_1", "existing_value"},
		},
		"flags and no args": {
			flagsInCtx: map[string]string{
				"flag_1": "some_value",
				"flag_2": "other_value",
				"flag_3": "abc",
			},
			givenCommand: []string{"some", "command"},
			givenFlags:   []string{"flag_2"},
			expected:     []string{"some", "command", "--flag_2", "other_value"},
		},
		"flags and args not overlaping": {
			flagsInCtx: map[string]string{
				"flag_1": "some_value",
				"flag_2": "other_value",
				"flag_3": "abc",
			},
			givenCommand: []string{"some", "command"},
			givenArgs:    []string{"--flag_1", "existing_value"},
			givenFlags:   []string{"flag_2"},
			expected:     []string{"some", "command", "--flag_2", "other_value", "--flag_1", "existing_value"},
		},
		"flags found in context but not in args": {
			flagsInCtx: map[string]string{
				"flag_1": "some_value",
				"flag_2": "other_value",
				"flag_3": "abc",
			},
			givenCommand: []string{"some", "command"},
			givenFlags:   []string{"flag_1", "flag_3"},
			expected:     []string{"some", "command", "--flag_1", "some_value", "--flag_3", "abc"},
		},
		"flag found in context and in args": {
			flagsInCtx: map[string]string{
				"flag_1": "some_value",
				"flag_2": "other_value",
				"flag_3": "abc",
			},
			givenCommand: []string{"some", "command"},
			givenArgs:    []string{"--flag_1", "existing_value"},
			givenFlags:   []string{"flag_1"},
			expected:     []string{"some", "command", "--flag_1", "existing_value"},
		},
		"flag does not have value": {
			flagsInCtx:   map[string]string{},
			givenCommand: []string{"some", "command"},
			givenFlags:   []string{"flag_1"},
			expected:     []string{"some", "command"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			flagSet := flag.NewFlagSet("flags", flag.ExitOnError)
			flagSet.String("flag_1", "", "")
			flagSet.String("flag_2", "", "")
			flagSet.String("flag_3", "", "")
			c := cli.NewContext(nil, flagSet, nil)
			for name, value := range test.flagsInCtx {
				require.NoError(t, c.Set(name, value))
			}
			res := prepareCommand(c, test.givenCommand, test.givenArgs, test.givenFlags...)
			assert.Equal(t, test.expected, res)
		})
	}
}
