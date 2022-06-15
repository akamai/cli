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

func TestInstallPHP(t *testing.T) {
	tests := map[string]struct {
		givenDir  string
		givenVer  string
		init      func(*mocked)
		withError error
	}{
		"install PHP, custom version with composer.phar file": {
			givenDir: "testDir",
			givenVer: "7.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "php").Return("/test/php", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/php",
					Args: []string{"/test/php", "-v"},
				}).Return([]byte("PHP 8.0.2 (cli)"), nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.json")).Return(true, nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.phar")).Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/php",
					Args: []string{"/test/php", filepath.Join("testDir", "composer.phar"), "install"},
					Dir:  "testDir",
				}).Return([]byte(""), nil).Once()
			},
		},
		"composer.phar exec error": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "php").Return("/test/php", nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.json")).Return(true, nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.phar")).Return(true, nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/php",
					Args: []string{"/test/php", filepath.Join("testDir", "composer.phar"), "install"},
					Dir:  "testDir",
				}).Return([]byte(""), &exec.ExitError{}).Once()
			},
			withError: ErrPackageManagerExec,
		},
		"install PHP, custom version with composer": {
			givenDir: "testDir",
			givenVer: "7.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "php").Return("/test/php", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/php",
					Args: []string{"/test/php", "-v"},
				}).Return([]byte("PHP 8.0.2 (cli)"), nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.json")).Return(true, nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.phar")).Return(false, nil).Once()
				m.On("LookPath", "composer").Return("/test/composer", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/composer",
					Args: []string{"/test/composer", "install"},
					Dir:  "testDir",
				}).Return([]byte(""), nil).Once()
			},
		},
		"composer exec error": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "php").Return("/test/php", nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.json")).Return(true, nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.phar")).Return(false, nil).Once()
				m.On("LookPath", "composer").Return("/test/composer", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/composer",
					Args: []string{"/test/composer", "install"},
					Dir:  "testDir",
				}).Return([]byte(""), &exec.ExitError{}).Once()
			},
			withError: ErrPackageManagerExec,
		},
		"install PHP, custom version with composer.phar runtime": {
			givenDir: "testDir",
			givenVer: "7.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "php").Return("/test/php", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/php",
					Args: []string{"/test/php", "-v"},
				}).Return([]byte("PHP 8.0.2 (cli)"), nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.json")).Return(true, nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.phar")).Return(false, nil).Once()
				m.On("LookPath", "composer").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "composer.phar").Return("/test/phar", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/phar",
					Args: []string{"/test/phar", "install"},
					Dir:  "testDir",
				}).Return([]byte(""), nil).Once()
			},
		},
		"composer.phar runtime exec error": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "php").Return("/test/php", nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.json")).Return(true, nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.phar")).Return(false, nil).Once()
				m.On("LookPath", "composer").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "composer.phar").Return("/test/phar", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/phar",
					Args: []string{"/test/phar", "install"},
					Dir:  "testDir",
				}).Return([]byte(""), &exec.ExitError{}).Once()
			},
			withError: ErrPackageManagerExec,
		},
		"no package manager found": {
			givenDir: "testDir",
			givenVer: "7.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "php").Return("/test/php", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/php",
					Args: []string{"/test/php", "-v"},
				}).Return([]byte("PHP 8.0.2 (cli)"), nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.json")).Return(true, nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.phar")).Return(false, nil).Once()
				m.On("LookPath", "composer").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "composer.phar").Return("", fmt.Errorf("not found")).Once()
			},
			withError: ErrPackageManagerNotFound,
		},
		"default version, without composer": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "php").Return("/test/php", nil).Once()
				m.On("FileExists", filepath.Join("testDir", "composer.json")).Return(false, nil).Once()
			},
		},
		"no version found": {
			givenDir: "testDir",
			givenVer: "7.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "php").Return("/test/php", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/php",
					Args: []string{"/test/php", "-v"},
				}).Return([]byte(""), nil).Once()
			},
			withError: ErrRuntimeNoVersionFound,
		},
		"version too low": {
			givenDir: "testDir",
			givenVer: "7.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "php").Return("/test/php", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/php",
					Args: []string{"/test/php", "-v"},
				}).Return([]byte("PHP 6.0.2 (cli)"), nil).Once()
			},
			withError: ErrRuntimeMinimumVersionRequired,
		},
		"runtime not found": {
			givenDir: "testDir",
			givenVer: "7.0.0",
			init: func(m *mocked) {
				m.On("LookPath", "php").Return("", fmt.Errorf("not found")).Once()
			},
			withError: ErrRuntimeNotFound,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := new(mocked)
			test.init(m)
			l := langManager{m}
			err := l.installPHP(context.Background(), test.givenDir, test.givenVer)
			m.AssertExpectations(t)
			if test.withError != nil {
				assert.True(t, errors.Is(err, test.withError), "want: %s; got: %s", test.withError, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
