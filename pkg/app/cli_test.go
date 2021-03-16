package app

import (
	"context"
	"flag"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"os"
	"testing"
)

func TestCreateApp(t *testing.T) {
	term := terminal.Color()
	ctx := terminal.Context(context.Background(), term)
	app := CreateApp(ctx)
	assert.Equal(t, "akamai", app.Name)
	assert.Equal(t, "Akamai CLI", app.Usage)
	assert.Equal(t, version.Version, app.Version)
	assert.Equal(t, term, app.Writer)
	assert.Equal(t, term.Error(), app.ErrWriter)
	assert.True(t, app.EnableBashCompletion)
	assert.True(t, hasFlag(app, "bash"))
	assert.True(t, hasFlag(app, "zsh"))
	assert.True(t, hasFlag(app, "proxy"))
	assert.True(t, hasFlag(app, "daemon"))
	assert.NotNil(t, app.Before)

	set := flag.NewFlagSet("test", 0)
	set.String("proxy", "", "")
	cliCtx := cli.NewContext(app, set, nil)
	require.NoError(t, cliCtx.Set("proxy", "https://test.akamai.com"))
	err := app.Before(cliCtx)
	require.NoError(t, err)
	assert.NotNil(t, log.FromContext(cliCtx.Context))
	assert.Equal(t, "https://test.akamai.com", os.Getenv("HTTP_PROXY"))
	assert.Equal(t, "https://test.akamai.com", os.Getenv("http_proxy"))
	assert.Equal(t, "https://test.akamai.com", os.Getenv("HTTPS_PROXY"))
	assert.Equal(t, "https://test.akamai.com", os.Getenv("https_proxy"))
}

func hasFlag(app *cli.App, name string) bool {
	for _, f := range app.Flags {
		if f.Names()[0] == name {
			return true
		}
	}
	return false
}
