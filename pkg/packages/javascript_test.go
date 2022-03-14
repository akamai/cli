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

func TestInstallJavaScript(t *testing.T) {
	tests := map[string]struct {
		givenDir  string
		givenVer  string
		init      func(*mocked)
		withError error
	}{
		"custom version with yarn and npm": {
			givenDir: "testDir",
			givenVer: "7.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "node").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "nodejs").Return("/test/nodejs", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/nodejs",
					Args: []string{"/test/nodejs", "-v"},
				}).Return([]byte("v14.8.0"), nil).Once()
				m.On("FileExists", "testDir/yarn.lock").Return(true, nil).Once()
				m.On("LookPath", "yarn").Return("/test/yarn", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/yarn",
					Args: []string{"/test/yarn", "install"},
					Dir:  "testDir",
				}).Return(nil, nil).Once()
				m.On("FileExists", "testDir/package.json").Return(true, nil).Once()
				m.On("LookPath", "npm").Return("/test/npm", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/npm",
					Args: []string{"/test/npm", "install"},
					Dir:  "testDir",
				}).Return(nil, nil).Once()
			},
		},
		"default version no yarn.lock and package.json": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "node").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "nodejs").Return("/test/nodejs", nil).Once()
				m.On("FileExists", "testDir/yarn.lock").Return(false, nil).Once()
				m.On("FileExists", "testDir/package.json").Return(false, nil).Once()
			},
		},
		"runtime not found": {
			givenDir: "testDir",
			givenVer: "7.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "node").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "nodejs").Return("", fmt.Errorf("not found")).Once()
			},
			withError: ErrRuntimeNotFound,
		},
		"version not found": {
			givenDir: "testDir",
			givenVer: "7.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "node").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "nodejs").Return("/test/nodejs", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/nodejs",
					Args: []string{"/test/nodejs", "-v"},
				}).Return([]byte(""), nil).Once()
			},
			withError: ErrRuntimeNoVersionFound,
		},
		"version too low": {
			givenDir: "testDir",
			givenVer: "7.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "node").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "nodejs").Return("/test/nodejs", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/nodejs",
					Args: []string{"/test/nodejs", "-v"},
				}).Return([]byte("v1.2.3"), nil).Once()
			},
			withError: ErrRuntimeMinimumVersionRequired,
		},
		"yarn runtime not found": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "node").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "nodejs").Return("/test/nodejs", nil).Once()
				m.On("FileExists", "testDir/yarn.lock").Return(true, nil).Once()
				m.On("LookPath", "yarn").Return("", fmt.Errorf("not found")).Once()
			},
			withError: ErrPackageManagerNotFound,
		},
		"yarn install error": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "node").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "nodejs").Return("/test/nodejs", nil).Once()
				m.On("FileExists", "testDir/yarn.lock").Return(true, nil).Once()
				m.On("LookPath", "yarn").Return("/test/yarn", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/yarn",
					Args: []string{"/test/yarn", "install"},
					Dir:  "testDir",
				}).Return(nil, &exec.ExitError{}).Once()
			},
			withError: ErrPackageManagerExec,
		},
		"npm runtime not found": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "node").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "nodejs").Return("/test/nodejs", nil).Once()
				m.On("FileExists", "testDir/yarn.lock").Return(false, nil).Once()
				m.On("FileExists", "testDir/package.json").Return(true, nil).Once()
				m.On("LookPath", "npm").Return("", fmt.Errorf("not found")).Once()
			},
			withError: ErrPackageManagerNotFound,
		},
		"npm install error": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "node").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "nodejs").Return("/test/nodejs", nil).Once()
				m.On("FileExists", "testDir/yarn.lock").Return(false, nil).Once()
				m.On("FileExists", "testDir/package.json").Return(true, nil).Once()
				m.On("LookPath", "npm").Return("/test/npm", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/npm",
					Args: []string{"/test/npm", "install"},
					Dir:  "testDir",
				}).Return(nil, &exec.ExitError{}).Once()
			},
			withError: ErrPackageManagerExec,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := new(mocked)
			test.init(m)
			l := langManager{m}
			err := l.installJavaScript(context.Background(), test.givenDir, test.givenVer)
			m.AssertExpectations(t)
			if test.withError != nil {
				assert.True(t, errors.Is(err, test.withError), "want: %s; got: %s", test.withError, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
