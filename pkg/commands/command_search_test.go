package commands

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCmdSearch(t *testing.T) {
	tests := map[string]struct {
		args         []string
		responseFile string
		init         func(*terminal.Mock)
		withError    string
	}{
		"search and find packages based on criteria": {
			args:         []string{"test"},
			responseFile: "packages-response.json",
			init: func(m *terminal.Mock) {
				bold := color.New(color.FgWhite, color.Bold)
				m.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{5})

				m.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"Test CLI", color.BlueString("test-cli")}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Command:")+" %s %s\n", []interface{}{"test-cmd", "(aliases: test, abc)"}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Description:")+" %s\n\n", []interface{}{"test for highest score"}).
					Return().Once()

				m.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"Test no cmd match", color.BlueString("test-no-cmd-match")}).
					Return().Once()

				m.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"Test CLI", color.BlueString("cli-1")}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Command:")+" %s %s\n", []interface{}{"title-cmd", ""}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Description:")+" %s\n\n", []interface{}{"test for match on title"}).
					Return().Once()

				m.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"Some CLI", color.BlueString("cli-4")}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Command:")+" %s %s\n", []interface{}{"test", ""}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Description:")+" %s\n\n", []interface{}{"test for match on command name"}).
					Return().Once()

				m.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"Some CLI", color.BlueString("cli-2")}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Command:")+" %s %s\n", []interface{}{"desc-cmd", ""}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Description:")+" %s\n\n", []interface{}{"test - match on description"}).
					Return().Once()

				m.On("Printf", "\nInstall using \"%s\".\n", []interface{}{color.BlueString("%s install [package]", tools.Self())}).
					Return().Once()
			},
		},
		"no match": {
			args:         []string{"abc123"},
			responseFile: "packages-response.json",
			init: func(m *terminal.Mock) {
				m.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{0}).Return().Once()
			},
		},
		"invalid response json": {
			args:         []string{"abc123"},
			responseFile: "invalid-response.json",
			init:         func(m *terminal.Mock) {},
			withError:    "unable to fetch remote Package List (",
		},
		"no args passed": {
			args:      []string{},
			init:      func(m *terminal.Mock) {},
			withError: "You must specify one or more keywords",
		},
	}

	for name, test := range tests {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/cli/package-list.json", r.URL.String())
			assert.Equal(t, http.MethodGet, r.Method)
			pkgResponse, err := ioutil.ReadFile(fmt.Sprintf("./testdata/cli-search/%s", test.responseFile))
			require.NoError(t, err)
			_, err = w.Write(pkgResponse)
			assert.NoError(t, err)
		}))
		defer srv.Close()
		require.NoError(t, os.Setenv("AKAMAI_CLI_PACKAGE_REPO", srv.URL))
		require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
		t.Run(name, func(t *testing.T) {
			m := &mocked{&terminal.Mock{}, &config.Mock{}, nil, nil, nil}
			command := &cli.Command{
				Name:   "search",
				Action: cmdSearch,
			}
			app, ctx := setupTestApp(command, m)
			args := os.Args[0:1]
			args = append(args, "search")
			args = append(args, test.args...)

			test.init(m.term)
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
