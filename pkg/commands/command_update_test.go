package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	gogit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func TestCmdUpdate(t *testing.T) {
	tests := map[string]struct {
		args      []string
		init      func(*testing.T, *mocked)
		teardown  func(*testing.T)
		withError string
	}{
		"update specific package": {
			args: []string{"echo"},
			init: func(t *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", "testdata/.akamai-cli/src/cli-echo").Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{1}), nil).Once()
				m.gitRepo.On("CommitObject", plumbing.Hash{1}).Return(&object.Commit{}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()
				m.langManager.On("Install", "testdata/.akamai-cli/src/cli-echo",
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"echo"}).Return(nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
			},
		},
		"update all packages": {
			args: []string{},
			init: func(t *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", "testdata/.akamai-cli/src/cli-echo").Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{1}), nil).Once()
				m.gitRepo.On("CommitObject", plumbing.Hash{1}).Return(&object.Commit{}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()
				m.langManager.On("Install", "testdata/.akamai-cli/src/cli-echo",
					packages.LanguageRequirements{Go: "1.14.0"}, []string{"echo"}).Return(nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
			},
		},
		"command is up to date": {
			args: []string{"echo"},
			init: func(t *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo"}).Return().Once()

				m.gitRepo.On("Open", "testdata/.akamai-cli/src/cli-echo").Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("WarnOK").Return().Once()
				m.term.On("Writeln", []interface{}{color.CyanString("command \"echo\" already up-to-date")}).Return(0, nil).Once()
			},
		},
		"error installing package": {
			args: []string{"echo-invalid-json"},
			init: func(t *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo-invalid-json"}).Return().Once()

				m.gitRepo.On("Open", "testdata/.akamai-cli/src/cli-echo-invalid-json").Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{1}), nil).Once()
				m.gitRepo.On("CommitObject", plumbing.Hash{1}).Return(&object.Commit{}, nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Installing...", []interface{}(nil)).Return().Once()
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusFail).Return().Once()
				m.term.On("Writeln", mock.Anything).Return(0, nil).Once()
			},
			withError: "Unable to update command",
		},
		"error fetching commit by hash": {
			args: []string{"echo-invalid-json"},
			init: func(t *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo-invalid-json"}).Return().Once()

				m.gitRepo.On("Open", "testdata/.akamai-cli/src/cli-echo-invalid-json").Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{1}), nil).Once()
				m.gitRepo.On("CommitObject", plumbing.Hash{1}).Return(nil, fmt.Errorf("oops")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "Unable to fetch updates (oops)",
		},
		"error getting HEAD of repository after pull": {
			args: []string{"echo-invalid-json"},
			init: func(t *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo-invalid-json"}).Return().Once()

				m.gitRepo.On("Open", "testdata/.akamai-cli/src/cli-echo-invalid-json").Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(nil)
				m.gitRepo.On("Head").Return(nil, fmt.Errorf("oops")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "Unable to fetch updates (oops)",
		},
		"error pulling repository": {
			args: []string{"echo-invalid-json"},
			init: func(t *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo-invalid-json"}).Return().Once()

				m.gitRepo.On("Open", "testdata/.akamai-cli/src/cli-echo-invalid-json").Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Head").Return(plumbing.NewHashReference("", plumbing.Hash{0}), nil).Once()
				m.gitRepo.On("Pull", worktree).Return(fmt.Errorf("oops"))

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "Unable to fetch updates (oops)",
		},
		"error getting HEAD of repository before pull": {
			args: []string{"echo-invalid-json"},
			init: func(t *testing.T, m *mocked) {
				worktree := &gogit.Worktree{}
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo-invalid-json"}).Return().Once()

				m.gitRepo.On("Open", "testdata/.akamai-cli/src/cli-echo-invalid-json").Return(nil).Once()
				m.gitRepo.On("Worktree").Return(worktree, nil).Once()
				m.gitRepo.On("Head").Return(nil, fmt.Errorf("oops")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "Unable to fetch updates (oops)",
		},
		"error getting worktree": {
			args: []string{"echo-invalid-json"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo-invalid-json"}).Return().Once()

				m.gitRepo.On("Open", "testdata/.akamai-cli/src/cli-echo-invalid-json").Return(nil).Once()
				m.gitRepo.On("Worktree").Return(nil, fmt.Errorf("oops")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "unable to update, there an issue with the package repo: oops",
		},
		"error opening repository": {
			args: []string{"echo-invalid-json"},
			init: func(t *testing.T, m *mocked) {
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", `Attempting to update "%s" command...`, []interface{}{"echo-invalid-json"}).Return().Once()

				m.gitRepo.On("Open", "testdata/.akamai-cli/src/cli-echo-invalid-json").Return(fmt.Errorf("oops")).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Fail").Return().Once()
			},
			withError: "unable to update, there an issue with the package repo: oops",
		},
		"error finding executable": {
			args:      []string{"not-found"},
			init:      func(t *testing.T, m *mocked) {},
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
			m := &mocked{&terminal.Mock{}, &config.Mock{}, &git.Mock{}, &packages.Mock{}, nil}
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
