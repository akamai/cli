package app

import (
	"context"
	"flag"
	"os"
	"regexp"
	"testing"

	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
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
}

func TestCreateAppTemplate_EmptyCommandName(t *testing.T) {
	term := terminal.Color()
	ctx := terminal.Context(context.Background(), term)

	app := CreateAppTemplate(ctx, "", "Akamai CLI", "some description", version.Version)
	assert.Equal(t, "akamai", app.Name)
	assert.Equal(t, "Akamai CLI", app.Usage)
	assert.Equal(t, "some description", app.Description)
	assert.Equal(t, version.Version, app.Version)
	assert.Equal(t, term, app.Writer)
	assert.Equal(t, term.Error(), app.ErrWriter)
	assert.Equal(t, "Copyright (C) Akamai Technologies, Inc", app.Copyright)
	assert.True(t, app.EnableBashCompletion)
	assert.True(t, hasFlag(app, "edgerc"))
	assert.True(t, hasFlag(app, "section"))
	assert.True(t, hasFlag(app, "accountkey"))
}

func TestCreateAppTemplate_NonEmptyCommandName(t *testing.T) {
	term := terminal.Color()
	ctx := terminal.Context(context.Background(), term)

	app := CreateAppTemplate(ctx, "test", "Akamai CLI", "some description", version.Version)
	assert.Equal(t, "akamai-test", app.Name)
}

func TestCreateAppTemplate_NonEmptyCommandNameWithEnvAKAMAI_CLI(t *testing.T) {
	term := terminal.Color()
	ctx := terminal.Context(context.Background(), term)

	assert.NoError(t, os.Setenv("AKAMAI_CLI", ""))
	app := CreateAppTemplate(ctx, "test", "", "", "")
	assert.Equal(t, "akamai test", app.Name)
	assert.NoError(t, os.Unsetenv("AKAMAI_CLI"))
}

func TestVersion(t *testing.T) {
	term := terminal.Color()
	ctx := terminal.Context(context.Background(), term)
	_ = CreateApp(ctx)
	assert.Regexp(t, regexp.MustCompile(`^--version\s+Output CLI version \(default: false\)$`), cli.VersionFlag.String())
	assert.Len(t, cli.VersionFlag.Names(), 1)
	assert.Equal(t, cli.VersionFlag.Names()[0], "version")
}

func TestCreateAppProxy(t *testing.T) {
	tests := map[string]struct {
		proxyValue   string
		expectedEnvs map[string]string
	}{
		"no proxy": {
			expectedEnvs: map[string]string{
				"HTTP_PROXY":  "",
				"HTTPS_PROXY": "",
			},
		},
		"proxy set without protocol": {
			proxyValue: "test.akamai.com",
			expectedEnvs: map[string]string{
				"HTTP_PROXY":  "http://test.akamai.com",
				"HTTPS_PROXY": "http://test.akamai.com",
			},
		},
		"proxy set with http": {
			proxyValue: "http://test.akamai.com",
			expectedEnvs: map[string]string{
				"HTTP_PROXY":  "http://test.akamai.com",
				"HTTPS_PROXY": "http://test.akamai.com",
			},
		},
		"proxy set with https": {
			proxyValue: "https://test.akamai.com",
			expectedEnvs: map[string]string{
				"HTTP_PROXY":  "https://test.akamai.com",
				"HTTPS_PROXY": "https://test.akamai.com",
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			term := terminal.Color()
			ctx := terminal.Context(context.Background(), term)
			app := CreateApp(ctx)
			set := flag.NewFlagSet("test", 0)
			set.String("proxy", "", "")
			cliCtx := cli.NewContext(app, set, nil)
			if test.proxyValue != "" {
				require.NoError(t, cliCtx.Set("proxy", test.proxyValue))
			}
			err := app.Before(cliCtx)
			require.NoError(t, err)
			assert.NotNil(t, log.FromContext(cliCtx.Context))
			for k, v := range test.expectedEnvs {
				assert.Equal(t, v, os.Getenv(k))
				require.NoError(t, os.Unsetenv(k))
			}
		})
	}
}

func hasFlag(app *cli.App, name string) bool {
	for _, f := range app.Flags {
		if f.Names()[0] == name {
			return true
		}
	}
	return false
}
