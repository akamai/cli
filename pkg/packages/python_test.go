package packages

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallPython(t *testing.T) {
	tests := map[string]struct {
		givenDir  string
		givenVer  string
		init      func(*mocked)
		withError error
	}{
		"with version 3 and pip": {
			givenDir: "testDir",
			givenVer: "3.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("/test/python3", nil).Once()
				m.On("LookPath", "pip3").Return("/test/pip3", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/python3",
					Args: []string{"/test/python3", "--version"},
				}, true).Return([]byte("Python 3.1.0"), nil).Once()
				m.On("FileExists", "testDir/requirements.txt").Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/pip3",
					Args: []string{"/test/pip3", "install", "--user", "--ignore-installed", "-r", "requirements.txt"},
					Dir:  "testDir",
				}).Return(nil, nil).Once()
			},
		},
		"with version 2 and pip": {
			givenDir: "testDir",
			givenVer: "2.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "python2").Return("/test/python2", nil).Once()
				m.On("LookPath", "pip2").Return("/test/pip2", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/python2",
					Args: []string{"/test/python2", "--version"},
				}, true).Return([]byte("Python 2.1.0"), nil).Once()
				m.On("FileExists", "testDir/requirements.txt").Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/pip2",
					Args: []string{"/test/pip2", "install", "--user", "--ignore-installed", "-r", "requirements.txt"},
					Dir:  "testDir",
				}).Return(nil, nil).Once()
			},
		},
		"with default version and pip": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("/test/python3", nil).Once()
				m.On("LookPath", "pip3").Return("/test/pip3", nil).Once()
				m.On("FileExists", "testDir/requirements.txt").Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/pip3",
					Args: []string{"/test/pip3", "install", "--user", "--ignore-installed", "-r", "requirements.txt"},
					Dir:  "testDir",
				}).Return(nil, nil).Once()
			},
		},
		"with default version and no pip": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("/test/python3", nil).Once()
				m.On("LookPath", "pip3").Return("/test/pip3", nil).Once()
				m.On("FileExists", "testDir/requirements.txt").Return(false, nil).Once()
			},
		},
		"pip exec error": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("/test/python3", nil).Once()
				m.On("LookPath", "pip3").Return("/test/pip3", nil).Once()
				m.On("FileExists", "testDir/requirements.txt").Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/pip3",
					Args: []string{"/test/pip3", "install", "--user", "--ignore-installed", "-r", "requirements.txt"},
					Dir:  "testDir",
				}).Return(nil, &exec.ExitError{}).Once()
			},
			withError: ErrPackageManagerExec,
		},
		"version not found": {
			givenDir: "testDir",
			givenVer: "3.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("/test/python3", nil).Once()
				m.On("LookPath", "pip3").Return("/test/pip3", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/python3",
					Args: []string{"/test/python3", "--version"},
				}, true).Return([]byte(""), nil).Once()
			},
			withError: ErrRuntimeNoVersionFound,
		},
		"version too low": {
			givenDir: "testDir",
			givenVer: "3.0.5",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("/test/python3", nil).Once()
				m.On("LookPath", "pip3").Return("/test/pip3", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/python3",
					Args: []string{"/test/python3", "--version"},
				}, true).Return([]byte("Python 3.0.1"), nil).Once()
			},
			withError: ErrRuntimeMinimumVersionRequired,
		},
		"python bin not found": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "python2").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "python").Return("", fmt.Errorf("not found")).Once()
			},
			withError: ErrRuntimeNotFound,
		},
		"pip bin not found": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("/test/python3", nil).Once()
				m.On("LookPath", "pip3").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "pip2").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "pip").Return("", fmt.Errorf("not found")).Once()
			},
			withError: ErrPackageManagerNotFound,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := new(mocked)
			test.init(m)
			l := langManager{m}
			err := l.installPython(context.Background(), test.givenDir, test.givenVer)
			m.AssertExpectations(t)
			if test.withError != nil {
				assert.True(t, errors.Is(err, test.withError), "want: %s; got: %s", test.withError, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
