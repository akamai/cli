package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/akamai/cli/v2/pkg/git"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/packages"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestCommandsLocator(t *testing.T) {
	require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
	ctx := terminal.Context(context.Background(), terminal.New(terminal.DiscardWriter(), os.Stdin, os.Stderr))
	res := CommandLocator(ctx)
	for i := 0; i < len(res)-1; i++ {
		assert.True(t, strings.Compare(res[i].Name, res[i+1].Name) == -1)
	}
}

func TestCreateInstalledCommandsWarnsForInvalidPackageMetadata(t *testing.T) {
	cliHome := t.TempDir()
	packageDir := filepath.Join(cliHome, ".akamai-cli", "src", "cli-invalid")
	require.NoError(t, os.MkdirAll(packageDir, 0700))
	require.NoError(t, os.WriteFile(
		filepath.Join(packageDir, "cli.json"),
		[]byte(`{"commands":[{"name":"valid"},{"name":"invalid.name"}]}`),
		0600,
	))
	t.Setenv("AKAMAI_CLI_HOME", cliHome)

	var logs bytes.Buffer
	logger := slog.New(log.NewHandler(&logs, false, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx := log.NewContext(context.Background(), *logger)
	term := new(terminal.Mock)
	term.On("Writeln", mock.Anything).Return(0, nil).Once()
	ctx = terminal.Context(ctx, term)

	commands := createInstalledCommands(ctx, &git.MockRepo{}, &packages.Mock{})

	term.AssertExpectations(t)
	require.Len(t, commands, 1)
	assert.Equal(t, "valid", commands[0].Name)
	assert.Contains(t, logs.String(), "Skipping command in installed package")
	assert.Contains(t, logs.String(), fmt.Sprintf("%q", packageDir))
	assert.Contains(t, logs.String(), `invalid command name "invalid.name" in cli.json`)
}

func TestSubcommandsToCliCommands_packagePrefix(t *testing.T) {
	from := subcommands{
		Commands: []command{{
			Name:         "testCmd",
			AutoComplete: false,
		}},
		Requirements: packages.LanguageRequirements{Python: "3.0.0"},
		Action:       nil,
		Pkg:          "testPkg",
	}

	cmds := subcommandToCliCommands(from, &git.MockRepo{}, &packages.Mock{})

	for _, cmd := range cmds {
		assert.True(t, strings.HasPrefix(cmd.Aliases[0], fmt.Sprintf("%s/", from.Pkg)), "there should be an alias with the package prefix")
	}
}

func TestSubcommandsToCliCommandsForwardsHelpArgument(t *testing.T) {
	commands := subcommandToCliCommands(subcommands{
		Pkg:      "cli-dns",
		Commands: []command{{Name: "dns"}},
	}, &git.MockRepo{}, &packages.Mock{})

	var forwardedArgs []string
	commands[0].Action = func(_ context.Context, cmd *cli.Command) error {
		forwardedArgs = cmd.Args().Slice()
		return nil
	}

	app := &cli.Command{
		Name:     "akamai",
		Commands: commands,
	}
	require.NoError(t, app.Run(context.Background(), []string{"akamai", "dns", "help"}))
	assert.Equal(t, []string{"help"}, forwardedArgs)
}

func TestPassthruCommand(t *testing.T) {
	akaEchoBin := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-echo", "bin", "akamai-echo")
	akaEchoPythonBin := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-echo", "bin", "akamai-echo-python")
	cliEchoPythonRepo := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-echo-python")
	cliEchoRepo := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-echo")
	cliEchoPythonVe := filepath.Join("testdata", ".akamai-cli", "venv", "cli-echo-python")
	tests := map[string]struct {
		executable       []string
		init             func(*mocked)
		langRequirements packages.LanguageRequirements
		dirName          string
		withError        error
	}{
		"golang binary": {
			executable: []string{akaEchoBin},
			init: func(m *mocked) {
				m.langManager.On(
					"FinishExecution", packages.LanguageRequirements{Go: "1.15.0"},
					cliEchoRepo).Once()
				m.cmd.On("Run").Return(nil).Once()
			},
			langRequirements: packages.LanguageRequirements{Go: "1.15.0"},
			dirName:          cliEchoRepo,
		},
		"python 2": {
			executable: []string{akaEchoPythonBin},
			init: func(m *mocked) {
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Python: "2.7.10"}, cliEchoPythonRepo).Once()
				m.cmd.On("Run").Return(nil).Once()
			},
			langRequirements: packages.LanguageRequirements{Python: "2.7.10"},
			dirName:          cliEchoPythonRepo,
		},
		"python 3, ve exists": {
			executable: []string{akaEchoPythonBin},
			init: func(m *mocked) {
				m.langManager.On("FileExists", cliEchoPythonVe).Return(true, nil)
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Python: "3.0.0"}, cliEchoPythonRepo).Once()
				m.cmd.On("Run").Return(nil).Once()
			},
			langRequirements: packages.LanguageRequirements{Python: "3.0.0"},
			dirName:          cliEchoPythonRepo,
		},
		"python 3, ve does not exist": {
			executable: []string{akaEchoPythonBin},
			init: func(m *mocked) {
				m.langManager.On("FileExists", cliEchoPythonVe).Return(false, nil)
				m.langManager.On("PrepareExecution", packages.LanguageRequirements{Python: "3.0.0"}, cliEchoPythonRepo).Return(nil).Once()
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Python: "3.0.0"}, cliEchoPythonRepo).Once()
				m.cmd.On("Run").Return(nil).Once()
			},
			langRequirements: packages.LanguageRequirements{Python: "3.0.0"},
			dirName:          cliEchoPythonRepo,
		},
		"python 3, ve does not exist - error running the external command": {
			executable: []string{akaEchoPythonBin},
			init: func(m *mocked) {
				m.langManager.On("FileExists", cliEchoPythonVe).Return(false, nil)
				m.langManager.On("PrepareExecution", packages.LanguageRequirements{Python: "3.0.0"}, cliEchoPythonRepo).Return(nil).Once()
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Python: "3.0.0"}, cliEchoPythonRepo).Return().Once()
				m.cmd.On("Run").Return(&exec.ExitError{ProcessState: &os.ProcessState{}}).Once()
			},
			langRequirements: packages.LanguageRequirements{Python: "3.0.0"},
			dirName:          cliEchoPythonRepo,
			withError:        cli.Exit("", 0),
		},
		"python 3, ve does not exist - error preparing execution": {
			executable: []string{akaEchoPythonBin},
			init: func(m *mocked) {
				m.langManager.On("FileExists", cliEchoPythonVe).Return(false, nil)
				m.langManager.On("PrepareExecution", packages.LanguageRequirements{Python: "3.0.0"}, cliEchoPythonRepo).Return(packages.ErrPackageManagerExec).Once()
			},
			langRequirements: packages.LanguageRequirements{Python: "3.0.0"},
			dirName:          cliEchoPythonRepo,
			withError:        packages.ErrPackageManagerExec,
		},
		"python 3 - fs permission error to read VE": {
			executable: []string{akaEchoPythonBin},
			init: func(m *mocked) {
				m.langManager.On("FileExists", cliEchoPythonVe).Return(false, fs.ErrPermission)
			},
			langRequirements: packages.LanguageRequirements{Python: "3.0.0"},
			dirName:          cliEchoPythonRepo,
			withError:        fs.ErrPermission,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cliHome, cliHomeOK := os.LookupEnv("AKAMAI_CLI_HOME")
			if err := os.Setenv("AKAMAI_CLI_HOME", "testdata"); err != nil {
				return
			}
			defer func() {
				if cliHomeOK {
					_ = os.Setenv("AKAMAI_CLI_HOME", cliHome)
				} else {
					_ = os.Unsetenv("AKAMAI_CLI_HOME")
				}
			}()

			m := &mocked{langManager: &packages.Mock{}, cmd: &MockCmd{}}
			test.init(m)

			err := passthruCommand(context.Background(), m.cmd, m.langManager, test.langRequirements, test.dirName)

			m.cmd.AssertExpectations(t)
			m.langManager.AssertExpectations(t)

			if test.withError != nil {
				actualErr, actualIsExitCoder := err.(cli.ExitCoder)
				expectedErr, expectedIsExitCoder := test.withError.(cli.ExitCoder)

				if actualIsExitCoder && expectedIsExitCoder {
					assert.Equal(t, expectedErr.Error(), actualErr.Error())
					assert.Equal(t, expectedErr.ExitCode(), actualErr.ExitCode())
				} else {
					assert.True(t, errors.Is(err, test.withError), "wanted: '%s'; got: '%s'", test.withError.Error(), err.Error())
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}
