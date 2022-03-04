package commands

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

func TestCommandsLocator(t *testing.T) {
	require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
	res := CommandLocator(context.Background())
	for i := 0; i < len(res)-1; i++ {
		assert.True(t, strings.Compare(res[i].Name, res[i+1].Name) == -1)
	}
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

	cmds := subcommandToCliCommands(from, &git.Mock{}, &packages.Mock{})

	for _, cmd := range cmds {
		assert.True(t, strings.HasPrefix(cmd.Aliases[0], fmt.Sprintf("%s/", from.Pkg)), "there should be an alias with the package prefix")
	}
}

func TestPassthruCommand(t *testing.T) {
	tests := map[string]struct {
		executable       []string
		init             func(*mocked)
		langRequirements packages.LanguageRequirements
		dirName          string
		withError        error
	}{
		"golang binary": {
			executable: []string{"./testdata/.akamai-cli/src/cli-echo/bin/akamai-echo"},
			init: func(m *mocked) {
				m.langManager.On(
					"FinishExecution", packages.LanguageRequirements{Go: "1.15.0"},
					"./testdata/.akamai-cli/src/cli-echo/").Once()
				m.cmd.On("Run").Return(nil).Once()
			},
			langRequirements: packages.LanguageRequirements{Go: "1.15.0"},
			dirName:          "./testdata/.akamai-cli/src/cli-echo/",
		},
		"python 2": {
			executable: []string{"./testdata/.akamai-cli/src/cli-echo/bin/akamai-echo-python"},
			init: func(m *mocked) {
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Python: "2.7.10"}, "./testdata/.akamai-cli/src/cli-echo-python/").Once()
				m.cmd.On("Run").Return(nil).Once()
			},
			langRequirements: packages.LanguageRequirements{Python: "2.7.10"},
			dirName:          "./testdata/.akamai-cli/src/cli-echo-python/",
		},
		"python 3, ve exists": {
			executable: []string{"./testdata/.akamai-cli/src/cli-echo/bin/akamai-echo-python"},
			init: func(m *mocked) {
				m.langManager.On("FileExists", "testdata/.akamai-cli/venv/cli-echo-python").Return(true, nil)
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Python: "3.0.0"}, "./testdata/.akamai-cli/src/cli-echo-python/").Once()
				m.cmd.On("Run").Return(nil).Once()
			},
			langRequirements: packages.LanguageRequirements{Python: "3.0.0"},
			dirName:          "./testdata/.akamai-cli/src/cli-echo-python/",
		},
		"python 3, ve does not exist": {
			executable: []string{"./testdata/.akamai-cli/src/cli-echo/bin/akamai-echo-python"},
			init: func(m *mocked) {
				m.langManager.On("FileExists", "testdata/.akamai-cli/venv/cli-echo-python").Return(false, nil)
				m.langManager.On("PrepareExecution", packages.LanguageRequirements{Python: "3.0.0"}, "./testdata/.akamai-cli/src/cli-echo-python/").Return(nil).Once()
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Python: "3.0.0"}, "./testdata/.akamai-cli/src/cli-echo-python/").Once()
				m.cmd.On("Run").Return(nil).Once()
			},
			langRequirements: packages.LanguageRequirements{Python: "3.0.0"},
			dirName:          "./testdata/.akamai-cli/src/cli-echo-python/",
		},
		"python 3, ve does not exist - error running the external command": {
			executable: []string{"./testdata/.akamai-cli/src/cli-echo/bin/akamai-echo-python"},
			init: func(m *mocked) {
				m.langManager.On("FileExists", "testdata/.akamai-cli/venv/cli-echo-python").Return(false, nil)
				m.langManager.On("PrepareExecution", packages.LanguageRequirements{Python: "3.0.0"}, "./testdata/.akamai-cli/src/cli-echo-python/").Return(nil).Once()
				m.langManager.On("FinishExecution", packages.LanguageRequirements{Python: "3.0.0"}, "./testdata/.akamai-cli/src/cli-echo-python/").Return().Once()
				m.cmd.On("Run").Return(&exec.ExitError{ProcessState: &os.ProcessState{}}).Once()
			},
			langRequirements: packages.LanguageRequirements{Python: "3.0.0"},
			dirName:          "./testdata/.akamai-cli/src/cli-echo-python/",
			withError:        fmt.Errorf("wanted"),
		},
		"python 3, ve does not exist - error preparing execution": {
			executable: []string{"./testdata/.akamai-cli/src/cli-echo/bin/akamai-echo-python"},
			init: func(m *mocked) {
				m.langManager.On("FileExists", "testdata/.akamai-cli/venv/cli-echo-python").Return(false, nil)
				m.langManager.On("PrepareExecution", packages.LanguageRequirements{Python: "3.0.0"}, "./testdata/.akamai-cli/src/cli-echo-python/").Return(packages.ErrPackageManagerExec).Once()
			},
			langRequirements: packages.LanguageRequirements{Python: "3.0.0"},
			dirName:          "./testdata/.akamai-cli/src/cli-echo-python/",
			withError:        packages.ErrPackageManagerExec,
		},
		"python 3 - fs permission error to read VE": {
			executable: []string{"./testdata/.akamai-cli/src/cli-echo/bin/akamai-echo-python"},
			init: func(m *mocked) {
				m.langManager.On("FileExists", "testdata/.akamai-cli/venv/cli-echo-python").Return(false, fs.ErrPermission)
			},
			langRequirements: packages.LanguageRequirements{Python: "3.0.0"},
			dirName:          "./testdata/.akamai-cli/src/cli-echo-python/",
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
				assert.True(t, errors.As(err, &test.withError), "want: '%s'; got '%s'", test.withError.Error(), err.Error())
				return
			}
			assert.NoError(t, err)
		})
	}
}
