package commands

import (
	"fmt"
	"os"
	"testing"

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCmdUninstall(t *testing.T) {
	tests := map[string]struct {
		args      []string
		init      func(*testing.T, *mocked)
		withError string
	}{
		"uninstall command": {
			args: []string{"echo-uninstall"},
			init: func(t *testing.T, m *mocked) {
				mustCopyFile(t, "./testdata/.akamai-cli/src/cli-echo/cli.json", "./testdata/.akamai-cli/src/cli-echo-uninstall")
				mustCopyFile(t, "./testdata/.akamai-cli/src/cli-echo/bin/akamai-echo", "./testdata/.akamai-cli/src/cli-echo-uninstall/bin")
				err := os.Rename("./testdata/.akamai-cli/src/cli-echo-uninstall/bin/akamai-echo", "./testdata/.akamai-cli/src/cli-echo-uninstall/bin/akamai-echo-uninstall")
				require.NoError(t, err)
				err = os.Chmod("./testdata/.akamai-cli/src/cli-echo-uninstall/bin/akamai-echo-uninstall", 0755)
				require.NoError(t, err)

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to uninstall "echo-uninstall" command...`, []interface{}(nil)).Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
			},
		},
		"package does not contain cli.json": {
			args: []string{"echo-uninstall"},
			init: func(t *testing.T, m *mocked) {
				mustCopyFile(t, "./testdata/.akamai-cli/src/cli-echo/bin/akamai-echo", "./testdata/.akamai-cli/src/cli-echo-uninstall/bin")
				err := os.Rename("./testdata/.akamai-cli/src/cli-echo-uninstall/bin/akamai-echo", "./testdata/.akamai-cli/src/cli-echo-uninstall/bin/akamai-echo-uninstall")
				require.NoError(t, err)
				err = os.Chmod("./testdata/.akamai-cli/src/cli-echo-uninstall/bin/akamai-echo-uninstall", 0755)
				require.NoError(t, err)

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to uninstall "echo-uninstall" command...`, []interface{}(nil)).Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "unable to uninstall, was it installed using " + color.CyanString("\"akamai install\"") + "?",
		},
		"executable not found": {
			args: []string{"invalid"},
			init: func(t *testing.T, m *mocked) {
			},
			withError: fmt.Sprintf(`command "invalid" not found. Try "%s help"`, tools.Self()),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
			m := &mocked{&terminal.Mock{}, &config.Mock{}, &git.Mock{}, &packages.Mock{}, nil}
			command := &cli.Command{
				Name:   "uninstall",
				Action: cmdUninstall(m.langManager),
			}
			app, ctx := setupTestApp(command, m)
			defer func() {
				require.NoError(t, os.RemoveAll("./testdata/.akamai-cli/src/cli-echo-uninstall"))
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
