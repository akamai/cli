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
			init:         func(m *mocked) {},
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
		"python command, default version, python command found": {
			givenReqs: LanguageRequirements{
				Python: "*",
			},
			givenCmdExec: "test",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("", fmt.Errorf("not found"))
				m.On("LookPath", "python2").Return("", fmt.Errorf("not found"))
				m.On("LookPath", "python").Return("/test/python", nil)
			},
			expected: []string{"/test/python", "test"},
		},
		"python command, default version, not found": {
			givenReqs: LanguageRequirements{
				Python: "*",
			},
			givenCmdExec: "test",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("", fmt.Errorf("not found"))
				m.On("LookPath", "python2").Return("", fmt.Errorf("not found"))
				m.On("LookPath", "python").Return("", fmt.Errorf("not found"))
			},
			withError: true,
		},
		"python command, version 3, binary found": {
			givenReqs: LanguageRequirements{
				Python: "3.0.0",
			},
			givenCmdExec: "test",
			init: func(m *mocked) {
				m.On("LookPath", "python3").Return("", fmt.Errorf("not found"))
				m.On("LookPath", "python").Return("/test/python", nil)
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
				m.On("LookPath", "python").Return("", fmt.Errorf("not found"))
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
		"python command, version 2, not found": {
			givenReqs: LanguageRequirements{
				Python: "2.0.0",
			},
			givenCmdExec: "test",
			init: func(m *mocked) {
				m.On("LookPath", "python2").Return("", fmt.Errorf("not found"))
				m.On("LookPath", "python").Return("", fmt.Errorf("not found"))
				m.On("LookPath", "python3").Return("", fmt.Errorf("not found"))
			},
			withError: true,
		},
		"not supported language": {
			givenReqs: LanguageRequirements{
				Ruby: "1.2.3",
			},
			givenCmdExec: "test",
			init:         func(m *mocked) {},
			expected:     []string{"test"},
		},
		"undefined language": {
			givenReqs:    LanguageRequirements{},
			givenCmdExec: "test",
			init:         func(m *mocked) {},
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
