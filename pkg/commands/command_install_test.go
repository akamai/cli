package commands

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/akamai/cli/pkg/color"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	git2 "github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCmdInstall(t *testing.T) {
	cliTestCmdRepo := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-test-cmd")
	cliJSON := filepath.Join(".", "testdata", "repo", "cli.json")
	cliInvalidJSON := filepath.Join(".", "testdata", "repo_invalid_json", "cli.json")
	cliNoBinaryJSON := filepath.Join(".", "testdata", "repo_no_binary", "cli.json")

	cliTestCmdJSON := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-test-cmd", "cli.json")
	cliTestInvalidJSONRepo := filepath.Join(".", "testdata", ".akamai-cli", "src", "cli-test-invalid-json")
	tests := map[string]struct {
		args                 []string
		init                 func(*testing.T, *mocked, *httptest.Server)
		teardown             func(*testing.T)
		binaryResponseStatus int
		withError            string
	}{
		"install from official akamai repository, build from source": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked, h *httptest.Server) {

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch package configuration from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()

				h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					configJSON, err := os.ReadFile(cliNoBinaryJSON)
					require.NoError(t, err)
					_, err = w.Write(configJSON)
					require.NoError(t, err)
				}))
				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.gitRepo.On("Clone", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					"https://github.com/akamai/cli-test-cmd.git", false, m.term).Return(nil).Once().
					Run(func(args mock.Arguments) {
						mustCopyFile(t, cliNoBinaryJSON, cliTestCmdRepo)
					})
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}, []string{""}).Return(nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				// list all packages
				m.term.On("Writeln", mock.Anything).Return(0, nil)
				m.term.On("Printf", mock.Anything, []interface{}(nil)).Return().Times(11)
				m.term.On("Printf", mock.Anything, []interface{}{"aliases"}).Return().Twice()
				m.term.On("Printf", mock.Anything, []interface{}{"alias"}).Return().Once()
				m.term.On("Printf", mock.Anything, []interface{}{"commands.test help [command]"}).Return().Once()
				m.term.On("Printf", mock.Anything, mock.Anything).Return().Twice()
				m.term.On("Printf", mock.Anything).Return().Twice()
			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll(cliTestCmdRepo))
			},
		},
		"install from official akamai repository, build from source + ldflags": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked, h *httptest.Server) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch package configuration from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()

				h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					configJSON, err := os.ReadFile(cliNoBinaryJSON)
					require.NoError(t, err)
					_, err = w.Write(configJSON)
					require.NoError(t, err)
				}))
				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.gitRepo.On("Clone", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					"https://github.com/akamai/cli-test-cmd.git", false, m.term).Return(nil).Once().
					Run(func(args mock.Arguments) {
						mustCopyFile(t, filepath.Join(".", "testdata", "repo_ldflags", "cli.json"), cliTestCmdRepo)
					})
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}, []string{"-X 'github.com/akamai/cli-test-command/cli.Version=1.0.0'"}).Return(nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				// list all packages
				m.term.On("Writeln", mock.Anything).Return(0, nil)
				m.term.On("Printf", mock.Anything, []interface{}(nil)).Return().Times(11)
				m.term.On("Printf", mock.Anything, []interface{}{"aliases"}).Return().Twice()
				m.term.On("Printf", mock.Anything, []interface{}{"alias"}).Return().Once()
				m.term.On("Printf", mock.Anything, []interface{}{"commands.test help [command]"}).Return().Once()
				m.term.On("Printf", mock.Anything, mock.Anything).Return().Twice()
				m.term.On("Printf", mock.Anything).Return().Twice()
			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll(cliTestCmdRepo))
			},
		},
		"install from official akamai repository, download binary": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked, h *httptest.Server) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch package configuration from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()

				h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					configJSON, err := os.ReadFile(cliJSON)
					output := strings.ReplaceAll(string(configJSON), "${REPOSITORY_URL}", os.Getenv("REPOSITORY_URL"))
					require.NoError(t, err)
					_, err = w.Write([]byte(output))
					require.NoError(t, err)

				}))
				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing Binaries...", []interface{}(nil)).Return().Once()
				m.term.On("OK").Return().Once()

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
			},
			binaryResponseStatus: http.StatusOK,
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll(cliTestCmdRepo))
			},
		},
		"package directory already exists": {
			args: []string{"installed"},
			init: func(t *testing.T, m *mocked, h *httptest.Server) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-installed.git"}).Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusWarn).Return().Once()
			},
			withError: color.RedString("Package directory already exists ("),
		},
		"no args passed": {
			args:      []string{},
			init:      func(t *testing.T, m *mocked, h *httptest.Server) {},
			withError: "You must specify a repository URL",
		},
		"git clone error": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked, h *httptest.Server) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch package configuration from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()

				h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					configJSON, err := os.ReadFile(cliNoBinaryJSON)
					require.NoError(t, err)
					_, err = w.Write(configJSON)
					require.NoError(t, err)
				}))
				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.gitRepo.On("Clone", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					"https://github.com/akamai/cli-test-cmd.git", false, m.term).Return(git.ErrPackageNotAvailable).Once().
					Run(func(args mock.Arguments) {
						mustCopyFile(t, cliJSON, cliTestCmdRepo)
					})
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()
				m.gitRepo.On("Clone", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(git2.ErrRepositoryAlreadyExists)
			},
			withError: "Package is not available. Supported packages can be found here: https://techdocs.akamai.com/home/page/products-tools-a-z",
		},
		"error reading downloaded package, invalid cli.json in repository": {
			args: []string{"test-invalid-json"},
			init: func(t *testing.T, m *mocked, h *httptest.Server) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch package configuration from %s...", []interface{}{"https://github.com/akamai/cli-test-invalid-json.git"}).Return().Once()

				h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					configJSON, err := os.ReadFile(cliNoBinaryJSON)
					require.NoError(t, err)
					_, err = w.Write(configJSON)
					require.NoError(t, err)
				}))
				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-invalid-json.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.gitRepo.On("Clone", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-invalid-json"),
					"https://github.com/akamai/cli-test-invalid-json.git", false, m.term).Return(nil).Once().
					Run(func(args mock.Arguments) {
						mustCopyFile(t, filepath.Join(".", "testdata", "repo_invalid_json", "cli.json"), cliTestInvalidJSONRepo)
					})
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll(cliTestInvalidJSONRepo))
			},
			withError: "Unable to install selected package",
		},
		"error reading downloaded package, invalid cli.json": {
			args: []string{"test-invalid-json"},
			init: func(t *testing.T, m *mocked, h *httptest.Server) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch package configuration from %s...", []interface{}{"https://github.com/akamai/cli-test-invalid-json.git"}).Return().Once()

				h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					configJSON, err := os.ReadFile(cliInvalidJSON)
					require.NoError(t, err)
					_, err = w.Write(configJSON)
					require.NoError(t, err)
				}))
				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-invalid-json.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll(cliTestInvalidJSONRepo))
			},
			withError: "Unable to install selected package",
		},
		"install from official akamai repository, unknown lang": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked, h *httptest.Server) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch package configuration from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()

				h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					configJSON, err := os.ReadFile(cliNoBinaryJSON)
					require.NoError(t, err)
					_, err = w.Write(configJSON)
					require.NoError(t, err)
				}))
				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.gitRepo.On("Clone", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					"https://github.com/akamai/cli-test-cmd.git", false, m.term).Return(nil).Once().
					Run(func(args mock.Arguments) {
						mustCopyFile(t, cliNoBinaryJSON, cliTestCmdRepo)
					})
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()
				m.langManager.On("Install", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}, []string{""}).Return(packages.ErrUnknownLang).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("WarnOK").Return().Once()

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll(cliTestCmdRepo))
			},
		},
		"install from official akamai repository, error downloading binary, invalid URL, build from source": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked, h *httptest.Server) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch package configuration from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()

				h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					configJSON, err := os.ReadFile(cliJSON)
					output := strings.ReplaceAll(string(configJSON), "${REPOSITORY_URL}", "invalid url")
					require.NoError(t, err)
					_, err = w.Write([]byte(output))
					require.NoError(t, err)

				}))
				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"

				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing Binaries...", []interface{}(nil)).Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusWarn)

				m.gitRepo.On("Clone", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					"https://github.com/akamai/cli-test-cmd.git", false, m.term).Return(nil).Once().
					Run(func(args mock.Arguments) {
						mustCopyFile(t, cliJSON, cliTestCmdRepo)
					})

				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}, []string{""}).Return(nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			binaryResponseStatus: http.StatusOK,
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll(cliTestCmdRepo))
			},
		},
		"install from official akamai repository, error downloading binary, invalid response status, build from source": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked, h *httptest.Server) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch package configuration from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()

				h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					configJSON, err := os.ReadFile(cliJSON)
					output := strings.ReplaceAll(string(configJSON), "${REPOSITORY_URL}", "invalid url")
					require.NoError(t, err)
					_, err = w.Write([]byte(output))
					require.NoError(t, err)

				}))
				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"

				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing Binaries...", []interface{}(nil)).Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusWarn)

				m.gitRepo.On("Clone", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					"https://github.com/akamai/cli-test-cmd.git", false, m.term).Return(nil).Once().
					Run(func(args mock.Arguments) {
						mustCopyFile(t, cliJSON, cliTestCmdRepo)
					})

				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}, []string{""}).Return(nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			binaryResponseStatus: http.StatusNotFound,
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll(cliTestCmdRepo))
			},
		},
		"error on install from source, binary does not exist": {
			args: []string{"test-cmd"},
			init: func(t *testing.T, m *mocked, h *httptest.Server) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch package configuration from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()

				h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					configJSON, err := os.ReadFile(cliNoBinaryJSON)
					output := strings.ReplaceAll(string(configJSON), "${REPOSITORY_URL}", "invalid url")
					require.NoError(t, err)
					_, err = w.Write([]byte(output))
					require.NoError(t, err)

				}))
				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing Binaries...", []interface{}(nil)).Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusWarn)

				m.gitRepo.On("Clone", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					"https://github.com/akamai/cli-test-cmd.git", false, m.term).Return(nil).Once().
					Run(func(args mock.Arguments) {
						mustCopyFile(t, filepath.Join(".", "testdata", "repo_no_binary", "cli.json"), cliTestCmdRepo)
						input, err := ioutil.ReadFile(cliTestCmdJSON)
						require.NoError(t, err)
						output := strings.ReplaceAll(string(input), "${REPOSITORY_URL}", os.Getenv("REPOSITORY_URL"))
						err = ioutil.WriteFile(cliTestCmdJSON, []byte(output), 0755)
						require.NoError(t, err)
					})
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-test-cmd.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()

				m.langManager.On("Install", filepath.Join("testdata", ".akamai-cli", "src", "cli-test-cmd"),
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"app-1-cmd-1"}, []string{""}).Return(fmt.Errorf("oops")).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusWarn).Return().Once()
				m.term.On("WriteError", "oops").Return(0, nil).Once()

				// list all packages
				m.term.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
				m.term.On("Writeln", mock.Anything).Return(0, nil)
			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll(cliTestCmdRepo))
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
			require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", filepath.Join(".", "testdata")))
			m := &mocked{&terminal.Mock{}, &config.Mock{}, &git.MockRepo{}, &packages.Mock{}, nil}
			h := &httptest.Server{}

			command := &cli.Command{
				Name:   "install",
				Action: cmdInstall(m.gitRepo, m.langManager),
			}
			app, ctx := setupTestApp(command, m)
			args := os.Args[0:1]
			args = append(args, "install")
			args = append(args, test.args...)

			test.init(t, m, h)
			if test.teardown != nil {
				defer test.teardown(t)
			}
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
