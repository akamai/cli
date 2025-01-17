package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/akamai/cli/pkg/color"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCmdSearch(t *testing.T) {
	tests := map[string]struct {
		args      []string
		init      func(*mocked)
		packages  *packageList
		withError string
	}{
		"search and find single package - sample when package is not installed": {
			args: []string{"sample"},
			init: func(m *mocked) {
				m.term.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{1})

				m.term.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"sample", color.BlueString("SAMPLE")}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Command:")+" %s %s\n", []interface{}{"sample", ""}).
					Return().Once()

				h := mockedServer("sample", "2.0.0", t)
				githubURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Printf", color.BoldString("  Available Version:")+" %s\n", []interface{}{"2.0.0"}).Return().Once()

				m.term.On("Printf", color.BoldString("  Description:")+" %s\n\n", []interface{}{"test for single match"}).
					Return().Once()
				m.term.On("Printf", "\nInstall using \"%s\".\n", []interface{}{color.BlueString("%s install [package]", tools.Self())}).
					Return().Once()
			},
			packages: packagesForTest,
		},
		"search and find single package - echo when installed version is less than available version": {
			args: []string{"echo-uninstall"},
			init: func(m *mocked) {
				m.term.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{1})

				m.term.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"echo", color.BlueString("echo")}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Command:")+" %s %s\n", []interface{}{"echo-uninstall", ""}).
					Return().Once()

				h := mockedServer("echo-uninstall", "2.0.0", t)
				githubURLTemplate = h.URL + "/akamai/%s/master/cli.json"

				m.term.On("Printf", color.BoldString("  Available Version:")+" %s\n", []interface{}{"2.0.0"}).Return().Once()

				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, "echo-uninstall").Return([]string{}, packages.ErrNoExeFound).Once()
				m.langManager.On("GetPackageBinPaths").Return("/path/to/echo-uninstall").Once()
				m.term.On("Printf", color.BoldString("  Installed Version:")+" %s\n", []interface{}{"1.0.0"}).Return().Once()

				m.term.On("Printf", color.BoldString("  Description:")+" %s\n\n", []interface{}{"test for single match"}).
					Return().Once()
				m.term.On("Printf", "\nUpdate using \"%s\".\n", []interface{}{color.BlueString("%s update [package]", tools.Self())}).
					Return().Once()
			},
			packages: packagesForTest,
		},
		"search and find single package - echo when  installed version is equal to available version": {
			args: []string{"echo-uninstall"},
			init: func(m *mocked) {
				m.term.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{1})

				m.term.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"echo", color.BlueString("echo")}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Command:")+" %s %s\n", []interface{}{"echo-uninstall", ""}).
					Return().Once()

				h := mockedServer("echo-uninstall", "1.0.0", t)
				githubURLTemplate = h.URL + "/akamai/%s/master/cli.json"

				m.term.On("Printf", color.BoldString("  Available Version:")+" %s\n", []interface{}{"1.0.0"}).Return().Once()

				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, "echo-uninstall").Return([]string{}, packages.ErrNoExeFound).Once()
				m.langManager.On("GetPackageBinPaths").Return("/path/to/echo-uninstall").Once()
				m.term.On("Printf", color.BoldString("  Installed Version:")+" %s\n", []interface{}{"1.0.0"}).Return().Once()

				m.term.On("Printf", color.BoldString("  Description:")+" %s\n\n", []interface{}{"test for single match"}).
					Return().Once()
				m.term.On("Printf", color.BlueString("Package is already up-to-date on your system"), []interface{}(nil)).
					Return().Once()
			},
			packages: packagesForTest,
		},
		"search and find multiple packages - cli": {
			args: []string{"cli"},
			init: func(m *mocked) {
				m.term.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{5})

				m.term.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"CLI no cmd match", color.BlueString("cli-no-cmd-match")}).
					Return().Once()

				m.term.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"abc-2", color.BlueString("cli-2")}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Command:")+" %s %s\n", []interface{}{"cli-2", "(aliases: abc, abc2)"}).
					Return().Once()

				h := mockedServer("cli-2", "1.0.0", t)
				githubURLTemplate = h.URL + "/akamai/%s/master/cli.json"

				m.term.On("Printf", color.BoldString("  Available Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Description:")+" %s\n\n", []interface{}{"test for match on name"}).
					Return().Once()

				m.term.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"cli-1", color.BlueString("abc-1")}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Command:")+" %s %s\n", []interface{}{"ClI-1", ""}).
					Return().Once()
				h = mockedServer("CLI-1", "1.0.0", t)
				githubURLTemplate = h.URL + "/akamai/%s/master/cli.json"

				m.term.On("Printf", color.BoldString("  Available Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Description:")+" %s\n\n", []interface{}{"test for match on title"}).
					Return().Once()

				m.term.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"abc-5", color.BlueString("abc-5")}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Command:")+" %s %s\n", []interface{}{"cli", ""}).
					Return().Once()
				h = mockedServer("cli", "1.0.0", t)
				githubURLTemplate = h.URL + "/akamai/%s/master/cli.json"

				m.term.On("Printf", color.BoldString("  Available Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Description:")+" %s\n\n", []interface{}{"test for match on command name"}).
					Return().Once()

				m.term.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"abc-3", color.BlueString("abc-3")}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Command:")+" %s %s\n", []interface{}{"abc-3", ""}).
					Return().Once()

				h = mockedServer("abc-3", "1.0.0", t)
				githubURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Printf", color.BoldString("  Available Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Description:")+" %s\n\n", []interface{}{"CLI - test for match on description"}).
					Return().Once()

				m.term.On("Printf", "\nInstall using \"%s\".\n", []interface{}{color.BlueString("%s install [package]", tools.Self())}).
					Return().Once()
			},
			packages: packagesForTest,
		},
		"search with no results - terraform": {
			args: []string{"terraform"},
			init: func(m *mocked) {
				m.term.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{0})
			},
			packages: packagesForTest,
		},
		"no args passed": {
			args:      []string{},
			init:      func(_ *mocked) {},
			withError: "You must specify one or more keywords",
		},
		"search and find single package - 404": {
			args: []string{"sample"},
			init: func(m *mocked) {
				m.term.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{1})
				m.term.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"sample", color.BlueString("SAMPLE")}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Command:")+" %s %s\n", []interface{}{"sample", ""}).
					Return().Once()

				h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					http.Error(w, "Not Found", http.StatusNotFound)

				}))
				githubURLTemplate = h.URL + "/akamai/%s/master/cli.json"
			},
			packages:  packagesForTest,
			withError: "error: status code 404",
		},
		"search and find single package - when no latest version found": {
			args: []string{"sample"},
			init: func(m *mocked) {
				m.term.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{1})

				m.term.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"sample", color.BlueString("SAMPLE")}).
					Return().Once()
				m.term.On("Printf", color.BoldString("  Command:")+" %s %s\n", []interface{}{"sample", ""}).
					Return().Once()

				h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					mockResponse := CLI{
						CommandList: []CommandObject{},
					}
					respBody, _ := json.Marshal(mockResponse)
					w.WriteHeader(http.StatusOK)
					var _, _ = w.Write(respBody)
				}))
				githubURLTemplate = h.URL + "/akamai/%s/master/cli.json"
			},
			withError: "no latest version found",
			packages:  packagesForTest,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", filepath.Join(".", "testdata")))
			m := &mocked{&terminal.Mock{}, &config.Mock{}, &git.MockRepo{}, &packages.Mock{}, nil}
			pr := &mockPackageReader{}
			pr.On("readPackage").Return(test.packages.copy(t), nil).Once()

			commandToExecute := &cli.Command{
				Name: "search",
				Action: func(context *cli.Context) error {
					return cmdSearchWithPackageReader(context, pr)
				},
			}

			app, ctx := setupTestApp(commandToExecute, m)
			args := os.Args[0:1]
			args = append(args, "search")
			args = append(args, test.args...)

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

// copy returns copy of packageList content
func (p *packageList) copy(t *testing.T) *packageList {
	bytes, err := json.Marshal(p)
	require.NoError(t, err)
	var pl packageList
	err = json.Unmarshal(bytes, &pl)
	require.NoError(t, err)

	return &pl
}

// packagesForTest is a package list with example packages
var packagesForTest = &packageList{
	Version: 1.0,
	Packages: []packageListItem{
		{
			Title: "cli-1",
			Name:  "abc-1",
			Commands: []command{
				{
					Name:        "ClI-1",
					Version:     "1.0.0",
					Description: "test for match on title",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title: "echo",
			Name:  "echo",
			Commands: []command{
				{
					Name:        "echo-uninstall",
					Version:     "1.0.0",
					Description: "test for single match",
				},
			},
			Requirements: requirements{
				Go: "1.14.0",
			},
		},
		{
			Title:   "abc-2",
			Name:    "cli-2",
			Version: "1.0.0",
			Commands: []command{
				{
					Name:        "cli-2",
					Aliases:     []string{"abc", "abc2"},
					Version:     "1.0.0",
					Description: "test for match on name",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title: "abc-3",
			Name:  "abc-3",
			Commands: []command{
				{
					Name:        "abc-3",
					Version:     "1.0.0",
					Description: "CLI - test for match on description",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title: "abc-4",
			Name:  "abc-4",
			Commands: []command{
				{
					Name:        "abc-4",
					Version:     "1.0.0",
					Description: "abc - no match",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title: "abc-5",
			Name:  "abc-5",
			Commands: []command{
				{
					Name:        "cli",
					Version:     "1.0.0",
					Description: "test for match on command name",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title: "CLI no cmd match",
			Name:  "cli-no-cmd-match",
			Commands: []command{
				{
					Name:        "abc-6",
					Version:     "1.0.0",
					Description: "title and name match, but no match on command",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title: "sample",
			Name:  "SAMPLE",
			Commands: []command{
				{
					Name:        "sample",
					Version:     "2.0.0",
					Description: "test for single match",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
	},
}

func mockedServer(name, version string, t *testing.T) *httptest.Server {
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mockResponse := CLI{
			CommandList: []CommandObject{
				{
					Name:    name,
					Version: version,
				},
			},
		}
		respBody, err := json.Marshal(mockResponse)
		if err != nil {
			t.Errorf("Error marshalling the response: %v", err)
			t.Fail()
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(respBody)
		if err != nil {
			t.Errorf("Error writing the response: %v", err)
			t.Fail()
		}

	}))
	return h
}
