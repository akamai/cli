package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPackage(t *testing.T) {
	tests := map[string]struct {
		directory string
		pkg       string
		withError string
	}{
		"return subcommands with directory name": {
			directory: "./testdata/repo",
			pkg:       "repo",
		},
		"return subcommands with directory name - strip prefix": {
			directory: "./testdata/.akamai-cli/src/cli-echo-python",
			pkg:       "echo-python",
		},
		"no error if no cli.json": {
			directory: "./testdata/cli-search",
			withError: `does not contain a cli.json`,
		},
		"return error if cli.json is not valid": {
			directory: "./testdata/.akamai-cli/src/cli-echo-invalid-json",
			withError: `invalid`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			subcommands, err := readPackage(test.directory)
			if test.withError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.withError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.pkg, subcommands.Pkg, "the package name was not resolved properly")
		})
	}
}
