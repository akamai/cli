package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/config"
	"github.com/akamai/cli/v2/pkg/git"
	"github.com/akamai/cli/v2/pkg/packages"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/tools"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCmdUpdate(t *testing.T) {
	cliEchoRepo := filepath.Join("testdata", ".akamai-cli", "src", "cli-echo")
	cliEchoBin := filepath.Join("testdata", ".akamai-cli", "src", "cli-echo", "bin", "akamai-echo")
	tempTestDir := filepath.Join(".", "testdata", "temp")
	tests := map[string]struct {
		args      []string
		init      func(*testing.T, *mocked)
		teardown  func(*testing.T)
		withError string
	}{
		"update specific package": {
			args: []string{"echo"},
			init: func(_ *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Reset", &gogit.ResetOptions{Mode: gogit.HardReset}).Return(nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{1}), nil).Once()
				m.gitRepo.On("CommitObject", plumbing.Hash{1}).Return(&object.Commit{}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing Dependencies...", []interface{}(nil)).Return().Once()
				m.langManager.On("Install", cliEchoRepo,
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"echo"}, []string{""}).Return(nil).Once()
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
			},
		},
		"update all packages": {
			args: []string{},
			init: func(_ *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Reset", &gogit.ResetOptions{Mode: gogit.HardReset}).Return(nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{1}), nil).Once()
				m.gitRepo.On("CommitObject", plumbing.Hash{1}).Return(&object.Commit{}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing Dependencies...", []interface{}(nil)).Return().Once()
				m.langManager.On("Install", cliEchoRepo,
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"echo"}, []string{""}).Return(nil).Once()
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
			},
		},
		"command is up to date": {
			args: []string{"echo"},
			init: func(_ *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Reset", &gogit.ResetOptions{Mode: gogit.HardReset}).Return(nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(fmt.Errorf("Unable to fetch updates (%w)", gogit.NoErrAlreadyUpToDate))
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("WarnOK").Return().Once()
				m.term.On("Writeln", []interface{}{color.CyanString("command \"echo\" already up-to-date")}).Return(0, nil).Once()
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()

				// installing update
				m.term.On("Start", `Installing Dependencies...`, []interface{}(nil)).Return().Once()
				m.langManager.On("Install", cliEchoRepo,
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"echo"}, []string{""}).Return(nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
			},
		},
		"error checking out master, continue normally": {
			args: []string{"echo"},
			init: func(_ *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Reset", &gogit.ResetOptions{Mode: gogit.HardReset}).Return(fmt.Errorf("an error")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Warn").Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Writeln", []interface{}{color.YellowString("unable to reset the branch changes, we will try to continue anyway: %s", "an error")}).Return(0, nil).Once()

				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{1}), nil).Once()
				m.gitRepo.On("CommitObject", plumbing.Hash{1}).Return(&object.Commit{}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing Dependencies...", []interface{}(nil)).Return().Once()
				m.langManager.On("Install", cliEchoRepo,
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"echo"}, []string{""}).Return(nil).Once()
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
			},
		},
		"error installing package": {
			args: []string{"echo"},
			init: func(_ *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Reset", &gogit.ResetOptions{Mode: gogit.HardReset}).Return(nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{1}), nil).Once()
				m.gitRepo.On("CommitObject", plumbing.Hash{1}).Return(&object.Commit{}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing Dependencies...", []interface{}(nil)).Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.langManager.On("Install", cliEchoRepo,
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"echo"}, []string{""}).Return(fmt.Errorf("oops")).Once()
				m.term.On("WriteError", "oops").Return(0, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "Unable to update command",
		},
		"error fetching commit by hash": {
			args: []string{"echo"},
			init: func(_ *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}

				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Reset", &gogit.ResetOptions{Mode: gogit.HardReset}).Return(nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{1}), nil).Once()
				m.gitRepo.On("CommitObject", plumbing.Hash{1}).Return(nil, fmt.Errorf("oops")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "Unable to fetch updates: oops",
		},
		"error getting HEAD of repository after pull": {
			args: []string{"echo"},
			init: func(_ *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Reset", &gogit.ResetOptions{Mode: gogit.HardReset}).Return(nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(nil, fmt.Errorf("oops")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "Unable to fetch updates: oops",
		},
		"error pulling repository": {
			args: []string{"echo"},
			init: func(_ *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Reset", &gogit.ResetOptions{Mode: gogit.HardReset}).Return(nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(git.ErrPackageNotAvailable)

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "Package is not available. Supported packages can be found here: https://techdocs.akamai.com/home/page/products-tools-a-z",
		},
		"error getting HEAD of repository before pull": {
			args: []string{"echo"},
			init: func(_ *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Reset", &gogit.ResetOptions{Mode: gogit.HardReset}).Return(nil).Once()
				m.gitRepo.On("Head").Return(nil, fmt.Errorf("oops")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "Unable to fetch updates: oops",
		},
		"error getting worktree": {
			args: []string{"echo"},
			init: func(_ *testing.T, m *mocked) {
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(nil).Once()
				m.gitRepo.On("Worktree").Return(nil, fmt.Errorf("oops")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "unable to update, there was an issue with the package repo: oops",
		},
		"error opening repository, up to date with remote": {
			args: []string{"echo"},
			init: func(t *testing.T, m *mocked) {
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()

				h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					configJSON, err := os.ReadFile(filepath.Join(cliEchoRepo, "cli.json"))
					require.NoError(t, err)
					_, err = w.Write(configJSON)
					require.NoError(t, err)
				}))

				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(fmt.Errorf("oops")).Once()

				m.term.On("Writeln", []interface{}{color.CyanString("command \"echo\" already up-to-date")}).Return(0, nil).Once()
				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("WarnOK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
			},
		},
		"error opening repository, update from remote, success": {
			args: []string{"echo"},
			init: func(t *testing.T, m *mocked) {

				mustCopyDirectory(t, cliEchoRepo, tempTestDir)

				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()
				configJSON, err := os.ReadFile(filepath.Join(cliEchoRepo, "cli.json"))
				require.NoError(t, err)
				h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					output := strings.ReplaceAll(string(configJSON), "1.0.0", "9.9.9")
					_, err = w.Write([]byte(output))
					require.NoError(t, err)
				}))

				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(fmt.Errorf("oops")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to fetch package configuration from %s...`, []interface{}{"https://github.com/akamai/cli-echo.git"}).Return().Once()
				m.term.On("OK").Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-echo.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.gitRepo.On("Clone", filepath.Join("testdata", ".akamai-cli", "src", "cli-echo"),
					"https://github.com/akamai/cli-echo.git", false, m.term).Return(nil).Once().
					Run(func(_ mock.Arguments) {
						mustCopyFile(t, filepath.Join(tempTestDir, "cli.json"), cliEchoRepo)
					})
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing Dependencies...", []interface{}(nil)).Return().Once()
				m.langManager.On("Install", filepath.Join("testdata", ".akamai-cli", "src", "cli-echo"),
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"echo"}, []string{""}).Return(nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll(cliEchoRepo))
				require.NoError(t, os.Rename(tempTestDir, cliEchoRepo))

			},
		},
		"error opening repository, update from remote, fail": {
			args: []string{"echo"},
			init: func(t *testing.T, m *mocked) {

				mustCopyDirectory(t, cliEchoRepo, tempTestDir)

				m.langManager.On("FindExec", packages.LanguageRequirements{Go: "1.14.0"}, cliEchoBin).Return([]string{cliEchoBin}, nil).Once()
				configJSON, err := os.ReadFile(filepath.Join(cliEchoRepo, "cli.json"))
				require.NoError(t, err)
				h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					output := strings.ReplaceAll(string(configJSON), "1.0.0", "9.9.9")
					_, err = w.Write([]byte(output))
					require.NoError(t, err)
				}))

				githubRawURLTemplate = h.URL + "/akamai/%s/master/cli.json"
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", cliEchoRepo).Return(fmt.Errorf("oops")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to fetch package configuration from %s...`, []interface{}{"https://github.com/akamai/cli-echo.git"}).Return().Once()
				m.term.On("OK").Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Attempting to fetch command from %s...", []interface{}{"https://github.com/akamai/cli-echo.git"}).Return().Once()
				m.term.On("OK").Return().Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()

				m.gitRepo.On("Clone", filepath.Join("testdata", ".akamai-cli", "src", "cli-echo"),
					"https://github.com/akamai/cli-echo.git", false, m.term).Return(nil).Once().
					Run(func(_ mock.Arguments) {
						mustCopyFile(t, filepath.Join(tempTestDir, "cli.json"), cliEchoRepo)
					})
				m.term.On("OK").Return().Once()
				m.term.On("Spinner").Return(m.term).Once()

				m.langManager.On("Install", filepath.Join("testdata", ".akamai-cli", "src", "cli-echo"),
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"echo"}, []string{""}).Return(fmt.Errorf("oops")).Once()
				m.term.On("Start", "Installing Dependencies...", []interface{}(nil)).Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
				m.term.On("WriteError", "oops").Return(0, nil).Once()

			},
			teardown: func(t *testing.T) {
				require.NoError(t, os.RemoveAll(cliEchoRepo))
				require.NoError(t, os.Rename(tempTestDir, cliEchoRepo))
			},
			withError: "unable to update: Unable to install selected package",
		},
		"error finding executable": {
			args:      []string{"not-found"},
			init:      func(_ *testing.T, _ *mocked) {},
			withError: fmt.Sprintf("Command \"not-found\" not found. Try \"%s help\".\n", tools.Self()),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/akamai/cli-test-command/releases/download/1.0.0/akamai-app-1-cmd-1", r.URL.String())
				assert.Equal(t, http.MethodGet, r.Method)
				_, err := w.Write([]byte(`binary content`))
				assert.NoError(t, err)
			}))
			defer srv.Close()
			require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
			m := &mocked{&terminal.Mock{}, &config.Mock{}, &git.MockRepo{}, &packages.Mock{}, nil}
			command := &cli.Command{
				Name:   "update",
				Action: cmdUpdate(m.gitRepo, m.langManager),
			}
			app, ctx := setupTestApp(command, m)
			app.Commands = append(app.Commands, &cli.Command{
				Name:     "echo",
				Category: "Installed",
			})
			args := os.Args[0:1]
			args = append(args, "update")
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
