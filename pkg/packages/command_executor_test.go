package packages

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecCommand(t *testing.T) {
	// there is no echo.exe in windows
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}
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
	if runtime.GOOS == "windows" {
		assert.True(t, strings.HasSuffix(res, "\\go.exe"))
	} else {
		assert.True(t, strings.HasSuffix(res, "/go"))
	}
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
