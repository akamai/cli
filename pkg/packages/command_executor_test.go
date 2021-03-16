package packages

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os/exec"
	"testing"
)

func TestExecCommand(t *testing.T) {
	executor := defaultExecutor{}
	cmd := exec.Command("echo", "test")
	res, err := executor.ExecCommand(cmd)
	assert.NoError(t, err)
	assert.Equal(t, "test\n", string(res))
}

func TestLookPath(t *testing.T) {
	executor := defaultExecutor{}
	res, err := executor.LookPath("go")
	assert.NoError(t, err)
	assert.Contains(t, res, "/go")
}

func TestFileExists(t *testing.T) {
	tests := map[string]struct {
		path      string
		expected  bool
		withError bool
	}{
		"file exists": {
			path:     "./command_executor_test.go",
			expected: true,
		},
		"file does not exist": {
			path:     "./abc",
			expected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			executor := defaultExecutor{}
			res, err := executor.FileExists(test.path)
			if test.withError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected, res)
		})
	}
}
