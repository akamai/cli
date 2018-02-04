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
		{"1", "2", 1},
		{"1.1", "1.2", 1},
		{"3.0.0", "3.1.4", 1},
	}

	for _, tt := range versionTests {
		if result := versionCompare(tt.left, tt.right); result != tt.result {
			t.Errorf("versionCompare(%s, %s) => %d, wanted: %d", tt.left, tt.right, result, tt.result)
		}
	}
}

func TestGithubize(t *testing.T) {
	githubizeTests := []struct {
		repoName   string
		result  string
	}{
		{"property", "https://github.com/akamai/cli-property.git"},
		{"cli-property", "https://github.com/akamai/cli-property.git"},
		{"akamai/cli-property", "https://github.com/akamai/cli-property.git"},
		{"https://github.com/akamai/cli-property.git", "https://github.com/akamai/cli-property.git"},
		{"file:///local/repo/path", "file:///local/repo/path"},
		{"file:///local/repo/path.git", "file:///local/repo/path.git"},
		{"ssh://example.org:/repo/path", "example.org:/repo/path"},
	}

	for _, tt := range githubizeTests {
		if result := githubize(tt.repoName); result != tt.result {
			t.Errorf("githubize(%s) => %s, wanted: %s", tt.repoName, result, tt.result)
		}
	}
}
