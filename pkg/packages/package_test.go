package packages

import (
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mocked struct {
	mock.Mock
}

func TestLangManager_FindExec(t *testing.T) {
	tests := map[string]struct {
		givenReqs    LanguageRequirements
		givenCmdExec string
		init         func(*mocked)
		expected     []string
		withError    bool
	}{
		"golang command": {
			givenReqs: LanguageRequirements{
				Go: "1.14.0",
			},
			givenCmdExec: "test",
			init:         func(_ *mocked) {},
			expected:     []string{"test"},
		},
		"js command, node found": {
			givenReqs: LanguageRequirements{
				Node: "7.0.0",
			},
			givenCmdExec: "test",
			init: func(m *mocked) {
				m.On("LookPath", "node").Return("/test/node", nil)
			},
			expected: []string{"/test/node", "test"},
		},
		"js command, nodejs found": {
			givenReqs: LanguageRequirements{
				Node: "7.0.0",
			},
			givenCmdExec: "test",
			init: func(m *mocked) {
				m.On("LookPath", "node").Return("", fmt.Errorf("not found"))
				m.On("LookPath", "nodejs").Return("/test/nodejs", nil)
			},
			expected: []string{"/test/nodejs", "test"},
		},
		"python command, version 3, binary found": {
			givenReqs: LanguageRequirements{
				Python: "3.0.0",
			},
			givenCmdExec: "test",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("/test/python", nil)
			},
			expected: []string{"/test/python", "test"},
		},
		"python command, version 3, not found": {
			givenReqs: LanguageRequirements{
				Python: "3.0.0",
			},
			givenCmdExec: "test",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("", fmt.Errorf("not found"))
				m.On("LookPath", "python3.exe").Return("", fmt.Errorf("not found"))
				m.On("LookPath", "py.exe").Return("", fmt.Errorf("not found"))
			},
			withError: true,
		},
		"python command, version 2, binary found": {
			givenReqs: LanguageRequirements{
				Python: "2.0.0",
			},
			givenCmdExec: "test",
			init: func(m *mocked) {
				m.On("LookPath", "python2").Return("/test/python2", nil)
			},
			expected: []string{"/test/python2", "test"},
		},
		"python command, version 2, python3 not found": {
			givenReqs: LanguageRequirements{
				Python: "2.0.0",
			},
			givenCmdExec: "test",
			init: func(m *mocked) {
				m.On("LookPath", "python2").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "python2.exe").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "py.exe").Return("", fmt.Errorf("not found")).Once()
			},
			withError: true,
		},
		"python command, version 2, not found": {
			givenReqs: LanguageRequirements{
				Python: "2.0.0",
			},
			givenCmdExec: "test",
			init: func(m *mocked) {
				m.On("LookPath", "python2").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "python2.exe").Return("", fmt.Errorf("not found")).Once()
				m.On("LookPath", "py.exe").Return("", fmt.Errorf("not found")).Once()
			},
			withError: true,
		},
		"not supported language": {
			givenReqs: LanguageRequirements{
				Ruby: "1.2.3",
			},
			givenCmdExec: "test",
			init:         func(_ *mocked) {},
			expected:     []string{"test"},
		},
		"undefined language": {
			givenReqs:    LanguageRequirements{},
			givenCmdExec: "test",
			init:         func(_ *mocked) {},
			expected:     []string{"test"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := new(mocked)
			test.init(m)
			l := langManager{m}
			res, err := l.FindExec(context.Background(), test.givenReqs, test.givenCmdExec)
			m.AssertExpectations(t)
			if test.withError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected, res)
		})
	}
}

func (m *mocked) ExecCommand(cmd *exec.Cmd, withCombinedOutput ...bool) ([]byte, error) {
	var args mock.Arguments
	if len(withCombinedOutput) > 0 {
		args = m.Called(cmd, withCombinedOutput[0])
	} else {
		args = m.Called(cmd)
	}
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mocked) LookPath(file string) (string, error) {
	args := m.Called(file)
	return args.String(0), args.Error(1)
}

func (m *mocked) FileExists(path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}

func (m *mocked) GetOS() string {
	args := m.Called()
	return args.String(0)
}

func TestDetermineLangAndRequirements(t *testing.T) {
	tests := map[string]struct {
		reqs     LanguageRequirements
		language string
		version  string
	}{
		"undefined": {
			language: Undefined,
			version:  "",
		},
		"concrete Python version": {
			reqs:     LanguageRequirements{Python: "2.7.10"},
			language: Python,
			version:  "2.7.10",
		},
		"Python with wildcard": {
			reqs:     LanguageRequirements{Python: "3.0.*"},
			language: Python,
			version:  "3.0.0",
		},
		"Python with short wildcard": {
			reqs:     LanguageRequirements{Python: "3.*"},
			language: Python,
			version:  "3.0.0",
		},
		"Python pure wildcard": {
			reqs:     LanguageRequirements{Python: "*"},
			language: Python,
			version:  "0.0.0",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			language, version := determineLangAndRequirements(test.reqs)
			assert.Equal(t, test.language, language)
			assert.Equal(t, test.version, version)
		})
	}
}
