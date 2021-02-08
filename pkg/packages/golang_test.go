package packages

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os/exec"
	"testing"
)

func TestInstallGolang(t *testing.T) {
	tests := map[string]struct {
		givenDir      string
		givenVer      string
		givenCommands []string
		init          func(*mocked)
		withError     error
	}{
		"default version using go modules": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test"},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("FileExists", "testDir/go.sum").Return(true, nil)
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
		"default version using go modules, multiple commands": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test1", "test2"},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("FileExists", "testDir/go.sum").Return(true, nil)
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
		"default version using go modules, go modules runtime not found": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test1", "test2"},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil).Once()
				m.On("FileExists", "testDir/go.sum").Return(true, nil)
				m.On("LookPath", "go").Return("", fmt.Errorf("not found")).Once()
				m.On("FileExists", "testDir/glide.lock").Return(true, nil)
				m.On("LookPath", "glide").Return("/test/glide", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/glide",
					Args: []string{"/test/glide", "install"},
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
		"default version using go modules, go mod execution error": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test1", "test2"},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil).Once()
				m.On("FileExists", "testDir/go.sum").Return(true, nil)
				m.On("LookPath", "go").Return("/test/go", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "mod", "tidy"},
					Dir:  "testDir",
				}).Return(nil, &exec.ExitError{})
				m.On("FileExists", "testDir/glide.lock").Return(true, nil)
				m.On("LookPath", "glide").Return("/test/glide", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/glide",
					Args: []string{"/test/glide", "install"},
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
		"default version using glide": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test"},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("FileExists", "testDir/go.sum").Return(false, nil)
				m.On("FileExists", "testDir/glide.lock").Return(true, nil)
				m.On("LookPath", "glide").Return("/test/glide", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/glide",
					Args: []string{"/test/glide", "install"},
					Dir:  "testDir",
				}).Return(nil, nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "build", "-o", "akamai-test", "."},
					Dir:  "testDir",
				}).Return(nil, nil)
			},
		},
		"default version, unable to find go.sum and glide.lock": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test"},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("FileExists", "testDir/go.sum").Return(false, nil)
				m.On("FileExists", "testDir/glide.lock").Return(false, nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "build", "-o", "akamai-test", "."},
					Dir:  "testDir",
				}).Return(nil, nil)
			},
		},
		"glide runtime not found": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test"},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("FileExists", "testDir/go.sum").Return(false, nil)
				m.On("FileExists", "testDir/glide.lock").Return(true, nil)
				m.On("LookPath", "glide").Return("/test/glide", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/glide",
					Args: []string{"/test/glide", "install"},
					Dir:  "testDir",
				}).Return(nil, &exec.ExitError{})
			},
			withError: ErrPackageManagerExec,
		},
		"glide exec error": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test"},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("FileExists", "testDir/go.sum").Return(false, nil)
				m.On("FileExists", "testDir/glide.lock").Return(true, nil)
				m.On("LookPath", "glide").Return("", fmt.Errorf("not found"))
			},
			withError: ErrPackageManagerNotFound,
		},
		"selected version OK": {
			givenDir:      "testDir",
			givenVer:      "1.14.0",
			givenCommands: []string{"test"},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/go",
					Args: []string{"/test/go", "version"},
				}).Return([]byte("go version go1.15.0 darwin/amd64"), nil)
				m.On("FileExists", "testDir/go.sum").Return(true, nil)
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
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", fmt.Errorf("not found"))
			},
			withError: ErrRuntimeNotFound,
		},
		"default version using go modules, command execution error": {
			givenDir:      "testDir",
			givenVer:      "*",
			givenCommands: []string{"test"},
			init: func(m *mocked) {
				m.On("LookPath", "go").Return("/test/go", nil)
				m.On("FileExists", "testDir/go.sum").Return(true, nil)
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
			err := l.installGolang(context.Background(), test.givenDir, test.givenVer, test.givenCommands)
			m.AssertExpectations(t)
			if test.withError != nil {
				assert.True(t, errors.Is(err, test.withError), "want: %s; got: %s", test.withError, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
