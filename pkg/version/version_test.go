package version

import (
	"testing"

	"github.com/tj/assert"
)

func TestCompareVersion(t *testing.T) {
	tests := map[string]struct {
		left, right string
		expected    int
	}{
		"left is greater than right":                 {"1.0.1", "1.0.0", Greater},
		"left is less than right":                    {"0.9.0", "1.0.0", Smaller},
		"versions are equal":                         {"0.9.0", "0.9.0", Equals},
		"left version does not match semver syntax":  {"abc", "0.9.0", Error},
		"right version does not match semver syntax": {"1.0.0", "abc", Error},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			res := Compare(test.left, test.right)
			assert.Equal(t, test.expected, res)
		})
	}
}
