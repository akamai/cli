package app

import (
	"bytes"
	"context"
	"os"
	"regexp"
	"testing"

	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
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
	assert.True(t, app.EnableShellCompletion)
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
	assert.True(t, app.EnableShellCompletion)
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
	assert.Regexp(t, regexp.MustCompile(`^--version\s+Output CLI version$`), cli.VersionFlag.String())
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
			t.Setenv("HTTP_PROXY", "")
			t.Setenv("HTTPS_PROXY", "")

			term := terminal.Color()
			ctx := terminal.Context(context.Background(), term)
			app := CreateApp(ctx)
			app.Action = func(context.Context, *cli.Command) error { return nil }
			args := []string{"akamai"}
			if test.proxyValue != "" {
				args = append(args, "--proxy", test.proxyValue)
			}
			err := app.Run(ctx, args)
			require.NoError(t, err)
			for k, v := range test.expectedEnvs {
				assert.Equal(t, v, os.Getenv(k))
			}
		})
	}
}

func hasFlag(app *cli.Command, name string) bool {
	for _, f := range app.Flags {
		if f.Names()[0] == name {
			return true
		}
	}
	return false
}

func TestAutocompletionFlag(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	term := &terminal.Mock{}
	term.On("Error").Return(stderr)

	ctx := terminal.Context(context.Background(), terminal.Terminal(term))

	app := CreateApp(ctx)
	app.Writer = stdout
	app.ErrWriter = stderr

	err := app.Run(ctx, []string{"akamai", "--generate-shell-completion"})
	assert.NoError(t, err)

	assert.Empty(t, stderr.String())

	stdoutString := stdout.String()
	assert.Contains(t, stdoutString, "help\n")
	assert.Contains(t, stdoutString, "--edgerc\n")
	assert.Contains(t, stdoutString, "-e\n")
	assert.Contains(t, stdoutString, "--section\n")
	assert.Contains(t, stdoutString, "-s\n")
	assert.Contains(t, stdoutString, "--bash\n")
	assert.Contains(t, stdoutString, "--zsh\n")
}
