package commands

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

func TestReadInstalledPackageOmitsInvalidCommandDiagnostics(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "cli.json"),
		[]byte(`{"commands":[{"name":"valid"},{"name":"invalid.name"}]}`),
		0600,
	))

	packageData, err := readInstalledPackage(dir)

	require.NoError(t, err)
	require.Len(t, packageData.Commands, 1)
	assert.Equal(t, "valid", packageData.Commands[0].Name)
}

func TestReadPackageFromGithubRejectsTraversalCommandName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"commands":[{"name":"x/../../escaped"}]}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	_, err := readPackageFromGithub(server.URL, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid command name in cli.json")
}

func TestDownloadBinReplacesSymlinkedOutputWithoutFollowingIt(t *testing.T) {
	dir := t.TempDir()
	externalFile := filepath.Join(t.TempDir(), "external")
	require.NoError(t, os.WriteFile(externalFile, []byte("original"), 0600))
	binaryName := "akamai-test"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(dir, binaryName)
	if err := os.Symlink(externalFile, binaryPath); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	err := downloadBin(context.Background(), dir, command{Name: "test"})

	require.Error(t, err)
	content, readErr := os.ReadFile(externalFile)
	require.NoError(t, readErr)
	assert.Equal(t, "original", string(content))
	info, statErr := os.Lstat(binaryPath)
	require.NoError(t, statErr)
	assert.Zero(t, info.Mode()&os.ModeSymlink)
}

func TestDownloadBinRejectsInvalidCommandName(t *testing.T) {
	err := downloadBin(context.Background(), t.TempDir(), command{
		Name: "x/../../escaped",
		Bin:  "https://example.com/akamai-test",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must match")
}

func TestDownloadBinRejectsSymlinkedBinaryDirectory(t *testing.T) {
	externalDir := t.TempDir()
	externalFile := filepath.Join(externalDir, "akamai-test")
	require.NoError(t, os.WriteFile(externalFile, []byte("original"), 0600))

	dir := filepath.Join(t.TempDir(), "bin")
	if err := os.Symlink(externalDir, dir); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	err := downloadBin(context.Background(), dir, command{Name: "test"})

	require.EqualError(t, err, fmt.Sprintf("output directory %q must be a non-symlink directory", dir))
	content, readErr := os.ReadFile(externalFile)
	require.NoError(t, readErr)
	assert.Equal(t, "original", string(content))
}

func TestDownloadBinSetsExecutablePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support Unix executable permissions")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("binary"))
		require.NoError(t, err)
	}))
	defer server.Close()

	dir := t.TempDir()
	require.NoError(t, downloadBin(context.Background(), dir, command{
		Name: "test",
		Bin:  server.URL,
	}))

	info, err := os.Stat(filepath.Join(dir, "akamai-test"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0775), info.Mode().Perm())
}
