package packages

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallGolang(t *testing.T) {
	tests := map[string]struct {
		givenDir      string
		givenVer      string
		givenCommands []string
		givenLdFlags  []string
		init          func(*mocked)
		withError     error
	}{
		"default version using go modules": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test"},
			givenLdFlags:  []string{""},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("FileExists", filepath.Join("testDir", "go.sum")).Return(true, nil)
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "mod", "tidy"},
					Dir:  "testDir",
				}).Return(nil, nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "build", "-o", "akamai-test", "."},
					Dir:  "testDir",
				}).Return(nil, nil)
			},
		},
		"default version using go modules, ldFlags": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test"},
			givenLdFlags:  []string{"-X 'github.com/akamai/cli-test/cli.Version=0.1.0'"},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("FileExists", filepath.Join("testDir", "go.sum")).Return(true, nil)
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "mod", "tidy"},
					Dir:  "testDir",
				}).Return(nil, nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "build", "-o", "akamai-test", `-ldflags=-X 'github.com/akamai/cli-test/cli.Version=0.1.0'`, "."},
					Dir:  "testDir",
				}).Return(nil, nil)
			},
		},
		"default version using go modules, multiple commands": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test1", "test2"},
			givenLdFlags:  []string{"", ""},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("FileExists", filepath.Join("testDir", "go.sum")).Return(true, nil)
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "mod", "tidy"},
					Dir:  "testDir",
				}).Return(nil, nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "build", "-o", "akamai-test1", "./test1"},
					Dir:  "testDir",
				}).Return(nil, nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "build", "-o", "akamai-test2", "./test2"},
					Dir:  "testDir",
				}).Return(nil, nil)
			},
		},
		"go modules runtime not found - return an error as glide support is removed": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test1", "test2"},
			givenLdFlags:  []string{"", ""},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil).Once()
				m.On("LookPath", "go").Return("", fmt.Errorf("not found")).Once()
			},
			withError: ErrRuntimeNotFound,
		},
		"default version using go modules, go mod execution error": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test1", "test2"},
			givenLdFlags:  []string{"", ""},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil).Once()
				m.On("FileExists", filepath.Join("testDir", "go.sum")).Return(true, nil)
				m.On("LookPath", "go").Return("/test/go", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "mod", "tidy"},
					Dir:  "testDir",
				}).Return(nil, &exec.ExitError{})
			},
			withError: ErrPackageManagerExec,
		},
		"selected version OK": {
			givenDir:      "testDir",
			givenVer:      "1.14.0",
			givenCommands: []string{"test"},
			givenLdFlags:  []string{""},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "version"},
				}).Return([]byte("go version go1.15.0 darwin/amd64"), nil)
				m.On("FileExists", filepath.Join("testDir", "go.sum")).Return(true, nil)
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "mod", "tidy"},
					Dir:  "testDir",
				}).Return(nil, nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "build", "-o", "akamai-test", "."},
					Dir:  "testDir",
				}).Return(nil, nil)
			},
		},
		"selected version not found": {
			givenDir:      "testDir",
			givenVer:      "1.14.0",
			givenCommands: []string{"test"},
			givenLdFlags:  []string{""},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "version"},
				}).Return([]byte(""), nil)
			},
			withError: ErrRuntimeNoVersionFound,
		},
		"selected version too low": {
			givenDir:      "testDir",
			givenVer:      "1.14.0",
			givenCommands: []string{"test"},
			givenLdFlags:  []string{""},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "version"},
				}).Return([]byte("go version go1.13.0 darwin/amd64"), nil)
			},
			withError: ErrRuntimeMinimumVersionRequired,
		},
		"runtime not found": {
			givenDir:      "testDir",
			givenVer:      "1.14.0",
			givenCommands: []string{"test"},
			givenLdFlags:  []string{""},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", fmt.Errorf("not found"))
			},
			withError: ErrRuntimeNotFound,
		},
		"default version using go modules, command execution error": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test"},
			givenLdFlags:  []string{""},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("FileExists", filepath.Join("testDir", "go.sum")).Return(true, nil)
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "mod", "tidy"},
					Dir:  "testDir",
				}).Return(nil, nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "build", "-o", "akamai-test", "."},
					Dir:  "testDir",
				}).Return(nil, &exec.ExitError{})
			},
			withError: ErrPackageCompileFailure,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := new(mocked)
			test.init(m)
			l := langManager{m}
			err := l.installGolang(context.Background(), test.givenDir, test.givenVer, test.givenCommands, test.givenLdFlags)
			m.AssertExpectations(t)
			if test.withError != nil {
				assert.True(t, errors.Is(err, test.withError), "want: %s; got: %s", test.withError, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
