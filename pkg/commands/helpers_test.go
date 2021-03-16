package commands

import (
	"context"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/git"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type mocked struct {
	term        *terminal.Mock
	cfg         *config.Mock
	gitRepo     *git.Mock
	langManager *packages.Mock
}

func setupTestApp(command *cli.Command, m *mocked) (*cli.App, context.Context) {
	cli.OsExiter = func(rc int) {}
	ctx := terminal.Context(context.Background(), m.term)
	ctx = config.Context(ctx, m.cfg)
	app := cli.NewApp()
	app.Commands = []*cli.Command{command}
	return app, ctx
}

func copyFile(t *testing.T, src, dst string) {
	err := os.MkdirAll(dst, 0755)
	require.NoError(t, err)
	srcFile, err := os.Open(src)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, srcFile.Close())
	}()
	destFile, err := os.Create(filepath.Join(dst, filepath.Base(srcFile.Name())))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, destFile.Close())
	}()
	_, err = io.Copy(destFile, srcFile)
	require.NoError(t, err)
	err = destFile.Sync()
	require.NoError(t, err)
}
