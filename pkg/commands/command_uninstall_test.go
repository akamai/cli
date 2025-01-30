package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/config"
	"github.com/akamai/cli/v2/pkg/git"
	"github.com/akamai/cli/v2/pkg/packages"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCmdUninstall(t *testing.T) {
	cliEchoJSON := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-echo", "cli.json")
	cliEchoUninstallRepo := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-echo-uninstall")
	cliEchoBin := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-echo", "bin", "akamai-echo")
	cliEchoUninstallBinDir := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-echo-uninstall", "bin")
	cliEchoInUninstallBin := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-echo-uninstall", "bin", "akamai-echo")
	cliEchoUninstallBin := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-echo-uninstall", "bin", "akamai-echo-uninstall")
	cliEchoUninstallWinBin := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-echo-uninstall", "bin", "akamai-echo-uninstall.cmd")
	tests := map[string]struct {
		args      []string
		init      func(*testing.T, *mocked)
		withError string
	}{
		"uninstall command": {
			args: []string{"echo-uninstall"},
			init: func(t *testing.T, m *mocked) {
				mustCopyFile(t, cliEchoJSON, cliEchoUninstallRepo)
				mustCopyFile(t, cliEchoBin, cliEchoUninstallBinDir)
				err := os.Rename(cliEchoInUninstallBin, cliEchoUninstallBin)
				require.NoError(t, err)
				err = os.Chmod(cliEchoUninstallBin, 0755)
				require.NoError(t, err)

				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoUninstallBin).Return([]string{cliEchoUninstallBin}, nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to uninstall "echo-uninstall" command...`, []interface{}(nil)).Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
			},
		},
		"package does not contain cli.json": {
			args:      []string{"echo-uninstall"},
			init:      func(_ *testing.T, _ *mocked) {},
			withError: fmt.Sprintf(`command "echo-uninstall" not found. Try "%s help"`, tools.Self()),
		},
		"unable to uninstall": {
			args: []string{"echo-uninstall"},
			init: func(t *testing.T, m *mocked) {
				mustCopyFile(t, cliEchoJSON, cliEchoUninstallRepo)
				mustCopyFile(t, cliEchoBin, cliEchoUninstallBinDir)
				var err error
				if runtime.GOOS == "windows" {
					err = os.Rename(cliEchoInUninstallBin, cliEchoUninstallWinBin)
					require.NoError(t, err)
					err = os.Chmod(cliEchoUninstallWinBin, 0755)
					require.NoError(t, err)
				} else {
					err = os.Rename(cliEchoInUninstallBin, cliEchoUninstallBin)
					require.NoError(t, err)
					err = os.Chmod(cliEchoUninstallBin, 0755)
					require.NoError(t, err)
				}
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoUninstallBin).Return([]string{}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to uninstall "echo-uninstall" command...`, []interface{}(nil)).Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
				m.term.On("OK").Return().Once()
			},
			withError: "unable to uninstall, was it installed using " + color.CyanString("\"akamai install\"") + "?",
		},
		"uninstall command when executable not found": {
			args: []string{"echo-uninstall"},
			init: func(t *testing.T, m *mocked) {
				mustCopyFile(t, cliEchoJSON, cliEchoUninstallRepo)
				mustCopyFile(t, cliEchoBin, cliEchoUninstallBinDir)
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, "echo-uninstall").Return([]string{}, packages.ErrNoExeFound).Once()
				m.langManager.On("GetPackageBinPaths").Return("/path/to/echo-uninstall").Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to uninstall "echo-uninstall" command...`, []interface{}(nil)).Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				m.term.On("RemoveAll", "/path/to/echo-uninstall").Return(nil).Once()

			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", filepath.Join(".", "testdata")))
			m := &mocked{&terminal.Mock{}, &config.Mock{}, &git.MockRepo{}, &packages.Mock{}, nil}
			command := &cli.Command{
				Name:   "uninstall",
				Action: cmdUninstall(m.langManager),
			}
			app, ctx := setupTestApp(command, m)
			defer func() {
				require.NoError(t, os.RemoveAll(cliEchoUninstallRepo))
			}()
			args := os.Args[0:1]
			args = append(args, "uninstall")
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
