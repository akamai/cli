package commands

import (
	"fmt"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCmdInstall(t *testing.T) {
	tests := map[string]struct {
		args                 []string
		init                 func(*testing.T, *mocked)
		teardown             func(*testing.T)
		binaryResponseStatus int
		withError            string
	}{
		"install from official akamai repository, build from source": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.gitRepo.On("Clone", "testdata/.akamai-cli/src/cli-test-cmd",
					"https://github.com/akamai/cli-test-cmd.git", false, m.term, 1).Return(nil).Once().
					Run(func(args mock.Arguments) {
						copyFile(t, "./testdata/repo/cli.json", "./testdata/.akamai-cli/src/cli-test-cmd")
					})
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", "testdata/.akamai-cli/src/cli-test-cmd",
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}).Return(nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll("./testdata/.akamai-cli/src/cli-test-cmd"))
			},
		},
		"install from official akamai repository, download binary": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.gitRepo.On("Clone", "testdata/.akamai-cli/src/cli-test-cmd",
					"https://github.com/akamai/cli-test-cmd.git", false, m.term, 1).Return(nil).Once().
					Run(func(args mock.Arguments) {
						copyFile(t, "./testdata/repo/cli.json", "./testdata/.akamai-cli/src/cli-test-cmd")
						input, err := ioutil.ReadFile("./testdata/.akamai-cli/src/cli-test-cmd/cli.json")
						require.NoError(t, err)
						output := strings.ReplaceAll(string(input), "${REPOSITORY_URL}", os.Getenv("REPOSITORY_URL"))
						err = ioutil.WriteFile("./testdata/.akamai-cli/src/cli-test-cmd/cli.json", []byte(output), 0755)
						require.NoError(t, err)
					})
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", "testdata/.akamai-cli/src/cli-test-cmd",
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}).Return(fmt.Errorf("oops")).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusWarn).Return().Once()
				m.term.On("Writeln", []interface{}{color.CyanString("oops")}).Return(0, nil).Once()
				m.term.On("IsTTY").Return(true).Once()
				m.term.On("Confirm", "Binary command(s) found, would you like to download and install it?", true).Return(true, nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Downloading binary...", []interface{}(nil)).Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusOK).Return().Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			binaryResponseStatus: http.StatusOK,
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll("./testdata/.akamai-cli/src/cli-test-cmd"))
			},
		},
		"package already exists": {
			args: []string{"installed"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-installed.git"}).Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
			},
			withError: color.RedString("Package directory already exists ("),
		},
		"no args passed": {
			args:      []string{},
			init:      func(t *testing.T, m *mocked) {},
			withError: "You must specify a repository URL",
		},
		"git clone error": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()

				m.gitRepo.On("Clone", "testdata/.akamai-cli/src/cli-test-cmd",
					"https://github.com/akamai/cli-test-cmd.git", false, m.term, 1).Return(fmt.Errorf("oops")).Once().
					Run(func(args mock.Arguments) {
						copyFile(t, "./testdata/repo/cli.json", "./testdata/.akamai-cli/src/cli-test-cmd")
					})
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
			},
			withError: "Unable to clone repository: oops",
		},
		"error reading downloaded package, invalid cli.json": {
			args: []string{"test-invalid-json"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-invalid-json.git"}).Return().Once()
				m.gitRepo.On("Clone", "testdata/.akamai-cli/src/cli-test-invalid-json",
					"https://github.com/akamai/cli-test-invalid-json.git", false, m.term, 1).Return(nil).Once().
					Run(func(args mock.Arguments) {
						copyFile(t, "./testdata/repo_invalid_json/cli.json", "./testdata/.akamai-cli/src/cli-test-invalid-json")
					})
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll("./testdata/.akamai-cli/src/cli-test-invalid-json"))
			},
			withError: "Unable to install selected package",
		},
		"install from official akamai repository, unknown lang": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.gitRepo.On("Clone", "testdata/.akamai-cli/src/cli-test-cmd",
					"https://github.com/akamai/cli-test-cmd.git", false, m.term, 1).Return(nil).Once().
					Run(func(args mock.Arguments) {
						copyFile(t, "./testdata/repo/cli.json", "./testdata/.akamai-cli/src/cli-test-cmd")
					})
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", "testdata/.akamai-cli/src/cli-test-cmd",
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}).Return(packages.ErrUnknownLang).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("WarnOK").Return().Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll("./testdata/.akamai-cli/src/cli-test-cmd"))
			},
		},
		"install from official akamai repository, user does not install binary": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.gitRepo.On("Clone", "testdata/.akamai-cli/src/cli-test-cmd",
					"https://github.com/akamai/cli-test-cmd.git", false, m.term, 1).Return(nil).Once().
					Run(func(args mock.Arguments) {
						copyFile(t, "./testdata/repo/cli.json", "./testdata/.akamai-cli/src/cli-test-cmd")
						input, err := ioutil.ReadFile("./testdata/.akamai-cli/src/cli-test-cmd/cli.json")
						require.NoError(t, err)
						output := strings.ReplaceAll(string(input), "${REPOSITORY_URL}", os.Getenv("REPOSITORY_URL"))
						err = ioutil.WriteFile("./testdata/.akamai-cli/src/cli-test-cmd/cli.json", []byte(output), 0755)
						require.NoError(t, err)
					})
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", "testdata/.akamai-cli/src/cli-test-cmd",
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}).Return(fmt.Errorf("oops")).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusWarn).Return().Once()
				m.term.On("Writeln", []interface{}{color.CyanString("oops")}).Return(0, nil).Once()
				m.term.On("IsTTY").Return(true).Once()
				m.term.On("Confirm", "Binary command(s) found, would you like to download and install it?", true).Return(false, nil).Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll("./testdata/.akamai-cli/src/cli-test-cmd"))
			},
			withError: "Unable to install selected package",
		},
		"install from official akamai repository, error downloading binary, invalid URL": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.gitRepo.On("Clone", "testdata/.akamai-cli/src/cli-test-cmd",
					"https://github.com/akamai/cli-test-cmd.git", false, m.term, 1).Return(nil).Once().
					Run(func(args mock.Arguments) {
						copyFile(t, "./testdata/repo/cli.json", "./testdata/.akamai-cli/src/cli-test-cmd")
					})
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", "testdata/.akamai-cli/src/cli-test-cmd",
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}).Return(fmt.Errorf("oops")).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusWarn).Return().Once()
				m.term.On("Writeln", []interface{}{color.CyanString("oops")}).Return(0, nil).Once()
				m.term.On("IsTTY").Return(true).Once()
				m.term.On("Confirm", "Binary command(s) found, would you like to download and install it?", true).Return(true, nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Downloading binary...", []interface{}(nil)).Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			binaryResponseStatus: http.StatusOK,
			withError:            "Unable to install selected package",
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll("./testdata/.akamai-cli/src/cli-test-cmd"))
			},
		},
		"install from official akamai repository, error downloading binary, invalid response status": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.gitRepo.On("Clone", "testdata/.akamai-cli/src/cli-test-cmd",
					"https://github.com/akamai/cli-test-cmd.git", false, m.term, 1).Return(nil).Once().
					Run(func(args mock.Arguments) {
						copyFile(t, "./testdata/repo/cli.json", "./testdata/.akamai-cli/src/cli-test-cmd")
						input, err := ioutil.ReadFile("./testdata/.akamai-cli/src/cli-test-cmd/cli.json")
						require.NoError(t, err)
						output := strings.ReplaceAll(string(input), "${REPOSITORY_URL}", os.Getenv("REPOSITORY_URL"))
						err = ioutil.WriteFile("./testdata/.akamai-cli/src/cli-test-cmd/cli.json", []byte(output), 0755)
						require.NoError(t, err)
					})
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", "testdata/.akamai-cli/src/cli-test-cmd",
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}).Return(fmt.Errorf("oops")).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusWarn).Return().Once()
				m.term.On("Writeln", []interface{}{color.CyanString("oops")}).Return(0, nil).Once()
				m.term.On("IsTTY").Return(true).Once()
				m.term.On("Confirm", "Binary command(s) found, would you like to download and install it?", true).Return(true, nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Downloading binary...", []interface{}(nil)).Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			binaryResponseStatus: http.StatusNotFound,
			withError:            "Unable to install selected package",
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll("./testdata/.akamai-cli/src/cli-test-cmd"))
			},
		},
		"error on install from source, binary does not exist": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.gitRepo.On("Clone", "testdata/.akamai-cli/src/cli-test-cmd",
					"https://github.com/akamai/cli-test-cmd.git", false, m.term, 1).Return(nil).Once().
					Run(func(args mock.Arguments) {
						copyFile(t, "./testdata/repo_no_binary/cli.json", "./testdata/.akamai-cli/src/cli-test-cmd")
						input, err := ioutil.ReadFile("./testdata/.akamai-cli/src/cli-test-cmd/cli.json")
						require.NoError(t, err)
						output := strings.ReplaceAll(string(input), "${REPOSITORY_URL}", os.Getenv("REPOSITORY_URL"))
						err = ioutil.WriteFile("./testdata/.akamai-cli/src/cli-test-cmd/cli.json", []byte(output), 0755)
						require.NoError(t, err)
					})
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", "testdata/.akamai-cli/src/cli-test-cmd",
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}).Return(fmt.Errorf("oops")).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusWarn).Return().Once()
				m.term.On("Writeln", []interface{}{color.CyanString("oops")}).Return(0, nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll("./testdata/.akamai-cli/src/cli-test-cmd"))
			},
			withError: "Unable to install selected package",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/akamai/cli-test-command/releases/download/1.0.0/akamai-app-1-cmd-1", r.URL.String())
				assert.Equal(t, http.MethodGet, r.Method)
				w.WriteHeader(test.binaryResponseStatus)
				_, err := w.Write([]byte(`binary content`))
				assert.NoError(t, err)
			}))
			defer srv.Close()
			require.NoError(t, os.Setenv("REPOSITORY_URL", srv.URL))
			require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
			m := &mocked{&terminal.Mock{}, &config.Mock{}, &git.Mock{}, &packages.Mock{}}
			command := &cli.Command{
				Name:   "install",
				Action: cmdInstall(m.gitRepo, m.langManager),
			}
			app, ctx := setupTestApp(command, m)
			args := os.Args[0:1]
			args = append(args, "install")
			args = append(args, test.args...)

			test.init(t, m)
			err := app.RunContext(ctx, args)
			if test.teardown != nil {
				test.teardown(t)
			}

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
