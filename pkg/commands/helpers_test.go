package commands

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/akamai/cli/v2/pkg/config"
	"github.com/akamai/cli/v2/pkg/git"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/packages"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

var testFiles = map[string][]string{
	"cli-echo":              {"akamai-e", "akamai-e.cmd", "akamai-echo", "akamai-echo.cmd", "akamai-echo-cmd.cmd"},
	"cli-echo-invalid-json": {"akamai-echo-invalid-json", "akamai-echo-invalid-json.cmd"},
}

// TestMain prepares test binary used as cli command in tests and copies it to each directory specified in 'testFiles' variable
// The reason why binaries are not included in the repository is to make the tests pass on different operating systems
// After tests are executed, all generated binaries are removed
func TestMain(m *testing.M) {
	log := slog.New(log.NewHandler(os.Stderr, false, nil))
	binaryPath, err := buildTestBinary()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	for dir, files := range testFiles {
		for _, file := range files {
			targetDir := filepath.Join("testdata", ".akamai-cli", "src", dir, "bin")
			if err := copyFile(binaryPath, targetDir); err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
			if err := os.Rename(filepath.Join(targetDir, filepath.Base(binaryPath)), filepath.Join(targetDir, file)); err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
			if err := os.Chmod(filepath.Join(targetDir, file), 0755); err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
		}
	}
	exitCode := m.Run()
	if err := os.RemoveAll(binaryPath); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	for dir := range testFiles {
		targetDir := filepath.Join("testdata", ".akamai-cli", "src", dir, "bin")
		if err := os.RemoveAll(targetDir); err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
	}
	os.Exit(exitCode)
}

func buildTestBinary() (string, error) {
	bin, err := exec.LookPath("go")
	if err != nil {
		return "", err
	}
	sourcePath := filepath.Join("testdata", "example-binary.go")
	targetPath := filepath.Join("testdata", "example-binary")
	if runtime.GOOS == "windows" {
		targetPath = fmt.Sprintf("%s.exe", targetPath)
	}
	cmd := exec.Command(bin, "build", "-o", targetPath, "-ldflags", "-s -w", sourcePath)
	_, err = cmd.Output()
	return targetPath, err
}

type mocked struct {
	term        *terminal.Mock
	cfg         *config.Mock
	gitRepo     *git.MockRepo
	langManager *packages.Mock
	cmd         *MockCmd
}

func setupTestApp(command *cli.Command, m *mocked) (*cli.App, context.Context) {
	cli.OsExiter = func(_ int) {}
	ctx := terminal.Context(context.Background(), m.term)
	ctx = config.Context(ctx, m.cfg)
	app := cli.NewApp()
	app.Commands = []*cli.Command{command}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "edgerc",
			Usage:   "edgerc config path passed to executed commands, defaults to ~/.edgerc",
			Aliases: []string{"e"},
		},
		&cli.StringFlag{
			Name:    "section",
			Usage:   "edgerc section name passed to executed commands, defaults to 'default'",
			Aliases: []string{"s"},
		},
	}
	return app, ctx
}

func copyFile(src, dst string) error {
	log := slog.New(log.NewHandler(os.Stderr, false, nil))
	err := os.MkdirAll(dst, 0755)
	if err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			log.Error(err.Error())
		}
	}()
	destFile, err := os.Create(filepath.Join(dst, filepath.Base(srcFile.Name())))
	if err != nil {
		return err
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			log.Error(err.Error())
		}
	}()
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}
	return destFile.Sync()
}

func mustCopyFile(t *testing.T, src, dst string) {
	require.NoError(t, copyFile(src, dst))
}

func mustCopyDirectory(t *testing.T, src, dst string) {
	require.NoError(t, copyDirectory(src, dst))
}

func copyDirectory(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dst, entry.Name())

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		if fileInfo.Mode().IsDir() {
			perm := fileInfo.Mode().Perm()
			if err := createIfNotExists(destPath, perm); err != nil {
				return err
			}
			if err := copyDirectory(sourcePath, destPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(sourcePath, dst); err != nil {
				return err
			}
		}

	}
	return nil
}

func exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func createIfNotExists(dir string, perm os.FileMode) error {
	if exists(dir) {
		return nil
	}

	err := os.MkdirAll(dir, perm)
	return err
}
