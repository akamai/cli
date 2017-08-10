package main

import "testing"

func TestVersionCompare(t *testing.T) {
	versionTests := []struct {
		left   string
		right  string
		result int
	}{
		{"0.9.9", "1.0.0", 1},
		{"0.1.0", "0.2.0", 1},
		{"0.3.0", "0.3.1", 1},
		{"0.1.0", "0.1.0", 0},
		{"1.0.0", "0.9.9", -1},
		{"0.2.0", "0.1.0", -1},
		{"0.3.1", "0.3.0", -1},
	}

	for _, tt := range versionTests {
		if result := versionCompare(tt.left, tt.right); result != tt.result {
			t.Errorf("versionCompare(%s, %s) => %d, wanted: %d", tt.left, tt.right, result, tt.result)
		}
	}
}
