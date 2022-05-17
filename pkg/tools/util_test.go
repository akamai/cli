// Copyright 2018. Akamai Technologies, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"testing"

	"github.com/akamai/cli/pkg/version"
	"github.com/stretchr/testify/assert"
)

func TestVersionCompare(t *testing.T) {
	versionTests := []struct {
		left   string
		right  string
		result int
	}{
		{"0.9.9", "1.0.0", version.Smaller},
		{"0.1.0", "0.2.0", version.Smaller},
		{"0.3.0", "0.3.1", version.Smaller},
		{"0.1.0", "0.1.0", version.Equals},
		{"1.0.0", "0.9.9", version.Greater},
		{"0.2.0", "0.1.0", version.Greater},
		{"0.3.1", "0.3.0", version.Greater},
		{"1", "2", version.Smaller},
		{"1.1", "1.2", version.Smaller},
		{"3.0.0", "3.1.4", version.Smaller},
		{"1.1.0", "1.1.1", version.Smaller},
		{"1.1.0", "1.1.1-dev", version.Smaller},
		{"1.0.4", "1.1.1-dev", version.Smaller},
		{"1.1.3", "1.1.4-dev", version.Smaller},
	}

	for _, tt := range versionTests {
		if result := version.Compare(tt.left, tt.right); result != tt.result {
			t.Errorf("versionCompare(%s, %s) => %d, wanted: %d", tt.left, tt.right, result, tt.result)
		}
	}
}

func TestGithubize(t *testing.T) {
	githubizeTests := []struct {
		repoName string
		result   string
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
		if result := Githubize(tt.repoName); result != tt.result {
			t.Errorf("Githubize(%s) => %s, wanted: %s", tt.repoName, result, tt.result)
		}
	}
}

func TestCapitalizeFirstWord(t *testing.T) {
	capitalizeTests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a", "A"},
		{"abc def", "Abc def"},
		{"QWE RTY", "QWE RTY"},
		{"0", "0"},
		{"+", "+"},
		{" ", " "},
	}

	for _, testCase := range capitalizeTests {
		if result := CapitalizeFirstWord(testCase.input); result != testCase.expected {
			t.Errorf("CapitalizeFirstWord(%s) => %s, wanted: %s", testCase.input, result, testCase.expected)
		}
	}
}

func TestInsertAfterNthWord(t *testing.T) {
	tests := map[string]struct {
		dest     string
		val      string
		index    int
		expected string
	}{
		"insert in the middle": {
			dest:     "test string with some words",
			val:      "[inserted string] and",
			index:    3,
			expected: "test string with [inserted string] and some words",
		},
		"insert at last index": {
			dest:     "test string with some words",
			val:      "and [inserted string]",
			index:    5,
			expected: "test string with some words and [inserted string]",
		},
		"insert at index larger than size": {
			dest:     "test string with some words",
			val:      "and [inserted string]",
			index:    6,
			expected: "test string with some words and [inserted string]",
		},
		"insert at the beginning": {
			dest:     "test string with some words",
			val:      "[inserted string] and",
			index:    0,
			expected: "[inserted string] and test string with some words",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			res := InsertAfterNthWord(test.dest, test.val, test.index)
			assert.Equal(t, test.expected, res)
		})
	}
}
