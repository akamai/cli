package commands

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestReadPackageRejectsTraversalCommandName(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cli.json"), []byte(`{"commands":[{"name":"x/../../escaped"}]}`), 0600))

	_, err := readPackage(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid command name in cli.json")
}

func TestReadPackageFromGithubRejectsTraversalCommandName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"commands":[{"name":"x/../../escaped"}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	_, err := readPackageFromGithub(server.URL, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid command name in cli.json")
}

func TestDownloadBinRejectsTraversalCommandName(t *testing.T) {
	dir := t.TempDir()
	escapedPath := filepath.Join(filepath.Dir(dir), "escaped")

	err := downloadBin(context.Background(), dir, command{Name: "x/../../escaped"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid command name in cli.json")
	assert.NoFileExists(t, escapedPath)
}
