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

func TestInstallRuby(t *testing.T) {
	tests := map[string]struct {
		givenDir  string
		givenVer  string
		init      func(*mocked)
		withError error
	}{
		"custom version, with bundler": {
			givenDir: "testDir",
			givenVer: "2.0.1",
			init: func(m *mocked) {
				m.On("LookPath", "ruby").Return("/test/ruby", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/ruby",
					Args: []string{"/test/ruby", "-v"},
				}).Return([]byte("ruby 2.6.3p62 (2021-01-01)"), nil).Once()
				m.On("FileExists", "testDir/Gemfile").Return(true, nil).Once()
				m.On("LookPath", "bundle").Return("/test/bundle", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/bundle",
					Args: []string{"/test/bundle", "install"},
					Dir:  "testDir",
				}).Return(nil, nil).Once()
			},
		},
		"default version, no bundler": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "ruby").Return("/test/ruby", nil).Once()
				m.On("FileExists", "testDir/Gemfile").Return(false, nil).Once()
			},
		},
		"runtime not found": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "ruby").Return("", fmt.Errorf("not found")).Once()
			},
			withError: ErrRuntimeNotFound,
		},
		"no version found": {
			givenDir: "testDir",
			givenVer: "2.0.1",
			init: func(m *mocked) {
				m.On("LookPath", "ruby").Return("/test/ruby", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/ruby",
					Args: []string{"/test/ruby", "-v"},
				}).Return([]byte(""), nil).Once()
			},
			withError: ErrRuntimeNoVersionFound,
		},
		"version too low": {
			givenDir: "testDir",
			givenVer: "2.0.1",
			init: func(m *mocked) {
				m.On("LookPath", "ruby").Return("/test/ruby", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/ruby",
					Args: []string{"/test/ruby", "-v"},
				}).Return([]byte("ruby 2.0.0p62 (2021-01-01)"), nil).Once()
			},
			withError: ErrRuntimeMinimumVersionRequired,
		},
		"bundle exec not found": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "ruby").Return("/test/ruby", nil).Once()
				m.On("FileExists", "testDir/Gemfile").Return(true, nil).Once()
				m.On("LookPath", "bundle").Return("", fmt.Errorf("not found")).Once()
			},
			withError: ErrPackageManagerNotFound,
		},
		"bundle exec error": {
			givenDir: "testDir",
			givenVer: "*",
			init: func(m *mocked) {
				m.On("LookPath", "ruby").Return("/test/ruby", nil).Once()
				m.On("FileExists", "testDir/Gemfile").Return(true, nil).Once()
				m.On("LookPath", "bundle").Return("/test/bundle", nil).Once()
				m.On("ExecCommand", &exec.Cmd{
					Path: "/test/bundle",
					Args: []string{"/test/bundle", "install"},
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
			err := l.installRuby(context.Background(), test.givenDir, test.givenVer)
			m.AssertExpectations(t)
			if test.withError != nil {
				assert.True(t, errors.Is(err, test.withError), "want: %s; got: %s", test.withError, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
