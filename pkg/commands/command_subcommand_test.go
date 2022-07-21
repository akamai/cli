package commands

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
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
	akamaiEchoBin := filepath.Join("testdata", ".akamai-cli", "src", "cli-echo", "bin", "akamai-echo")
	akamaiEBin := filepath.Join("testdata", ".akamai-cli", "src", "cli-echo", "bin", "akamai-e")
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
				m.langManager.On("PrepareExecution", packages.LanguageRequirements{Go: "1.14.0"}, "cli-echo").Return(nil).Once()
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Go: "1.14.0"}, "cli-echo").Once()
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, akamaiEchoBin).Return([]string{akamaiEchoBin}, nil)
			},
		},
		"run installed akamai echo command as binary with edgerc location": {
			command: "echo",
			args:    []string{"abc"},
			init: func(t *testing.T, m *mocked) {
				m.langManager.On("PrepareExecution", packages.LanguageRequirements{Go: "1.14.0"}, "cli-echo").Return(nil).Once()
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Go: "1.14.0"}, "cli-echo").Once()
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, akamaiEchoBin).Return([]string{akamaiEchoBin}, nil)
			},
		},
		"run installed akamai echo command as binary with alias": {
			command:        "e",
			args:           []string{"abc"},
			edgercLocation: "some/location",
			section:        "some_section",
			init: func(t *testing.T, m *mocked) {
				m.langManager.On("PrepareExecution", packages.LanguageRequirements{Go: "1.14.0"}, "cli-echo").Return(nil).Once()
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Go: "1.14.0"}, "cli-echo").Once()
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, akamaiEBin).Return([]string{akamaiEBin}, nil)
			},
		},
		"run installed akamai echo command as .cmd file": {
			command: "echo-cmd",
			args:    []string{"abc"},
			init: func(t *testing.T, m *mocked) {
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, filepath.Join("testdata", ".akamai-cli", "src", "cli-echo", "bin", "akamai-echo-cmd.cmd")).
					Return([]string{filepath.Join("testdata", ".akamai-cli", "src", "cli-echo", "bin", "akamai-echo-cmd.cmd")}, nil)
				m.langManager.On("PrepareExecution", packages.LanguageRequirements{Go: "1.14.0"}, "cli-echo").Return(nil).Once()
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Go: "1.14.0"}, "cli-echo").Once()
			},
		},
		"run installed python akamai echo command": {
			command: "echo-cmd",
			args:    []string{"abc"},
			init: func(t *testing.T, m *mocked) {
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, filepath.Join("testdata", ".akamai-cli", "src", "cli-echo", "bin", "akamai-echo-cmd.cmd")).
					Return([]string{filepath.Join("testdata", ".akamai-cli", "src", "cli-echo", "bin", "akamai-echo-cmd.cmd")}, nil)
				m.langManager.On("PrepareExecution", packages.LanguageRequirements{Go: "1.14.0"}, "cli-echo").Return(nil).Once()
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Go: "1.14.0"}, "cli-echo").Once()
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
			m := &mocked{&terminal.Mock{}, &config.Mock{}, &git.MockRepo{}, &packages.Mock{}, nil}
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

func TestPythonCmdSubcommand(t *testing.T) {
	// run installed akamai echo command with python required
	t.Run("run installed akamai echo command with python required", func(t *testing.T) {
		require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
		m := &mocked{&terminal.Mock{}, &config.Mock{}, &git.MockRepo{}, &packages.Mock{}, nil}
		command := &cli.Command{
			Name:   "echo-python",
			Action: cmdSubcommand(m.gitRepo, m.langManager),
		}
		app, ctx := setupTestApp(command, m)
		args := os.Args[0:1]
		args = append(args, "echo-python")
		args = append(args, "abc")

		// Using the system python avoids the need of shipping a python
		// interpreter together with the test data. This wouldn't be a
		// good option because of different OSes and CPU architectures.
		if pythonBin, err := exec.LookPath("python"); err != nil {
			// If python is not available, just skip the test
			t.Skipf("We could not find any available Python binary, thus we skip this test. Details: \n%s", err.Error())
		} else {
			m.langManager.On("PrepareExecution", packages.LanguageRequirements{Python: "3.0.0"}, "cli-echo-python").Return(nil).Once()
			m.langManager.On("FinishExecution", packages.LanguageRequirements{Python: "3.0.0"}, "cli-echo-python").Return(nil).Once()
			m.langManager.On("FindExec", packages.LanguageRequirements{Python: "3.0.0"}, filepath.Join("testdata", ".akamai-cli", "src", "cli-echo-python")).
				Return([]string{pythonBin}, nil).Once()
			m.langManager.On("FileExists", filepath.Join("testdata", ".akamai-cli", "venv", "cli-echo-python")).Return(true, nil)
		}

		err := app.RunContext(ctx, args)

		m.cfg.AssertExpectations(t)
		require.NoError(t, err)
	})
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
			givenCommand: []string{"command"},
			expected:     []string{"command"},
		},
		"no flags and with args": {
			givenCommand: []string{"command"},
			givenArgs:    []string{"--flag_1", "existing_value"},
			expected:     []string{"command", "--flag_1", "existing_value"},
		},
		"flags and no args": {
			flagsInCtx: map[string]string{
				"flag_1": "some_value",
				"flag_2": "other_value",
				"flag_3": "abc",
			},
			givenCommand: []string{"command"},
			givenFlags:   []string{"flag_2"},
			expected:     []string{"command"},
		},
		"flags and args not overlaping": {
			flagsInCtx: map[string]string{
				"flag_1": "some_value",
				"flag_2": "other_value",
				"flag_3": "abc",
			},
			givenCommand: []string{"command"},
			givenArgs:    []string{"--flag_1", "existing_value"},
			givenFlags:   []string{"flag_2"},
			expected:     []string{"command", "--flag_2", "other_value", "--flag_1", "existing_value"},
		},
		"flags found in context but not in args": {
			flagsInCtx: map[string]string{
				"flag_1": "some_value",
				"flag_2": "other_value",
				"flag_3": "abc",
			},
			givenCommand: []string{"command"},
			givenFlags:   []string{"flag_1", "flag_3"},
			expected:     []string{"command"},
		},
		"flag found in context and in args": {
			flagsInCtx: map[string]string{
				"flag_1": "some_value",
				"flag_2": "other_value",
				"flag_3": "abc",
			},
			givenCommand: []string{"command"},
			givenArgs:    []string{"--flag_1", "existing_value"},
			givenFlags:   []string{"flag_1"},
			expected:     []string{"command", "--flag_1", "existing_value"},
		},
		"flag does not have value": {
			flagsInCtx:   map[string]string{},
			givenCommand: []string{"command"},
			givenFlags:   []string{"flag_1"},
			expected:     []string{"command"},
		},

		// tests for script commands like python or js
		"script - no flags and no args": {
			givenCommand: []string{"some", "command"},
			expected:     []string{"some", "command"},
		},
		"script - no flags and with args": {
			givenCommand: []string{"some", "command"},
			givenArgs:    []string{"--flag_1", "existing_value"},
			expected:     []string{"some", "command", "--flag_1", "existing_value"},
		},
		"script - flags and no args": {
			flagsInCtx: map[string]string{
				"flag_1": "some_value",
				"flag_2": "other_value",
				"flag_3": "abc",
			},
			givenCommand: []string{"some", "command"},
			givenFlags:   []string{"flag_2"},
			expected:     []string{"some", "command"},
		},
		"script - flags and args not overlaping": {
			flagsInCtx: map[string]string{
				"flag_1": "some_value",
				"flag_2": "other_value",
				"flag_3": "abc",
			},
			givenCommand: []string{"some", "command"},
			givenArgs:    []string{"--flag_1", "existing_value"},
			givenFlags:   []string{"flag_2"},
			expected:     []string{"some", "command", "--flag_1", "existing_value", "--flag_2", "other_value"},
		},
		"script - flags found in context but not in args": {
			flagsInCtx: map[string]string{
				"flag_1": "some_value",
				"flag_2": "other_value",
				"flag_3": "abc",
			},
			givenCommand: []string{"some", "command"},
			givenFlags:   []string{"flag_1", "flag_3"},
			expected:     []string{"some", "command"},
		},
		"script -  flag found in context and in args": {
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
		"script - flag does not have value": {
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
