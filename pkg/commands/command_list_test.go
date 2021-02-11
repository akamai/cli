package commands

import (
	"fmt"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCmdListWithRemote(t *testing.T) {
	tests := map[string]struct {
		args      []string
		init      func(*mocked)
		withError string
	}{
		"list all commands": {
			init: func(m *mocked) {
				bold := color.New(color.FgWhite, color.Bold)
				m.term.On("Writeln", []interface{}{color.YellowString("\nInstalled Commands:\n")}).Return(0, nil).Once()

				// List command
				m.term.On("Printf", bold.Sprintf("  list"), []interface{}(nil)).Return().Once()
				m.term.On("Printf", " (%s: ", []interface{}{"aliases"}).Return().Once()
				m.term.On("Printf", bold.Sprintf("ls"), []interface{}(nil)).Return().Once()
				m.term.On("Printf", ", ", []interface{}(nil)).Return().Once()
				m.term.On("Printf", bold.Sprintf("show"), []interface{}(nil)).Return().Once()
				m.term.On("Printf", ")", []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}(nil)).Return(0, nil).Once()
				m.term.On("Printf", "    %s\n", []interface{}{"Displays available commands"}).Return().Once()

				// Help command
				m.term.On("Printf", bold.Sprintf("  help"), []interface{}(nil)).Return().Once()
				m.term.On("Printf", " (%s: ", []interface{}{"alias"}).Return().Once()
				m.term.On("Printf", bold.Sprintf("h"), []interface{}(nil)).Return().Once()
				m.term.On("Printf", ")", []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}(nil)).Return(0, nil).Once()

				m.term.On("Printf", "\nSee \"%s\" for details.\n", []interface{}{color.BlueString("%s help [command]", tools.Self())}).Return().Once()

				m.term.On("Writeln", []interface{}{color.YellowString("\nAvailable Commands:\n\n")}).Return(0, nil).Once()
				m.term.On("Printf", bold.Sprint("  test-remote-command"), []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}{fmt.Sprintf(" [package: %s]", color.BlueString("remote-package"))}).Return(0, nil).Once()
				m.term.On("Printf", "    %s\n", []interface{}{"Test remote command"}).Return().Once()
				m.term.On("Printf", "\nInstall using \"%s\".\n", []interface{}{color.BlueString("%s install [package]", tools.Self())}).Return().Once()
			},
		},
	}

	for name, test := range tests {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/cli/package-list.json", r.URL.String())
			assert.Equal(t, http.MethodGet, r.Method)
			_, err := w.Write([]byte(`{"packages": [{"name":"remote-package","commands": [{"name":"test-remote-command","description":"Test remote command"}]}]}`))
			assert.NoError(t, err)
		}))
		defer srv.Close()
		require.NoError(t, os.Setenv("AKAMAI_CLI_PACKAGE_REPO", srv.URL))
		require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
		t.Run(name, func(t *testing.T) {
			m := &mocked{&terminal.Mock{}, &config.Mock{}, nil, nil}
			command := &cli.Command{
				Name: "list",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "remote",
					},
				},
				Description: "Displays available commands",
				Aliases:     []string{"ls", "show"},
				Action:      cmdList,
			}
			app, ctx := setupTestApp(command, m)
			args := os.Args[0:1]
			args = append(args, "list", "--remote")

			test.init(m)
			err := app.RunContext(ctx, args)

			m.cfg.AssertExpectations(t)
			m.term.AssertExpectations(t)
			if test.withError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.withError)
				return
			}
			require.NoError(t, err)
		})
	}
}
