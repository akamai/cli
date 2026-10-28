package autocomplete

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestComplete_root_cmd(t *testing.T) {
	outbuf := &bytes.Buffer{}
	app := &cli.Command{
		Name:                  "akamai",
		EnableShellCompletion: true,
		ShellComplete:         Default,
		HideHelp:              true,
		Writer:                outbuf,
		Commands: []*cli.Command{
			{
				Name:          "test",
				ShellComplete: Default,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "flag",
					},
				},
			},
			{
				Name:          "test2",
				ShellComplete: Default,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "flag2",
					},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "gflag1",
			},
			&cli.StringFlag{
				Name: "gflag2",
			},
		},
	}

	complete(t, app, "akamai", "--generate-shell-completion")

	assert.Equal(t, "test\ntest2\n--gflag1\n--gflag2\n", outbuf.String())
}

func TestComplete_subcommand(t *testing.T) {
	outbuf := &bytes.Buffer{}
	app := &cli.Command{
		Name:                  "akamai",
		EnableShellCompletion: true,
		HideHelp:              true,
		Writer:                outbuf,
		Commands: []*cli.Command{
			{
				Name:          "test",
				ShellComplete: Default,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "flag",
					},
				},
				Commands: []*cli.Command{
					{
						Name: "sub1",
					},
					{
						Name:   "sub2",
						Hidden: true,
					},
				},
			},
		},
	}

	complete(t, app, "akamai", "test", "--generate-shell-completion")

	assert.Equal(t, "sub1\n--flag\n", outbuf.String())
}

func TestComplete_short_flag(t *testing.T) {
	outbuf := &bytes.Buffer{}
	app := &cli.Command{
		Name:                  "akamai",
		EnableShellCompletion: true,
		HideHelp:              true,
		Writer:                outbuf,
		Commands: []*cli.Command{
			{
				Name:          "test",
				ShellComplete: Default,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "flag",
						Aliases: []string{"f"},
					},
				},
			},
		},
	}

	complete(t, app, "akamai", "test", "--generate-shell-completion")

	assert.Equal(t, "--flag\n-f\n", outbuf.String())
}

func TestComplete_hides_help_flag_for_hidden_help_command(t *testing.T) {
	originalHelpFlag := cli.HelpFlag
	cli.HelpFlag = &cli.BoolFlag{Name: "help", Aliases: []string{"h"}}
	t.Cleanup(func() { cli.HelpFlag = originalHelpFlag })

	outbuf := &bytes.Buffer{}
	app := &cli.Command{
		Name:                  "akamai",
		EnableShellCompletion: true,
		ShellComplete:         Default,
		Writer:                outbuf,
		Commands: []*cli.Command{
			{
				Name:            "config",
				HideHelpCommand: true,
				ShellComplete:   Default,
				Flags: []cli.Flag{
					cli.HelpFlag,
					&cli.StringFlag{Name: "setting"},
				},
			},
		},
	}

	complete(t, app, "akamai", "config", "--generate-shell-completion")

	assert.Equal(t, "--setting\n", outbuf.String())
}

func TestComplete_root_omits_help_alias(t *testing.T) {
	originalHelpFlag := cli.HelpFlag
	cli.HelpFlag = &cli.BoolFlag{Name: "help", Aliases: []string{"h"}}
	t.Cleanup(func() { cli.HelpFlag = originalHelpFlag })

	outbuf := &bytes.Buffer{}
	app := &cli.Command{
		Name:                  "akamai",
		EnableShellCompletion: true,
		ShellComplete:         Default,
		Writer:                outbuf,
		Flags:                 []cli.Flag{cli.HelpFlag},
	}

	complete(t, app, "akamai", "--generate-shell-completion")

	assert.Equal(t, "help\n--help\n", outbuf.String())
}

func TestComplete_help_command(t *testing.T) {
	outbuf := &bytes.Buffer{}
	app := &cli.Command{
		Name:                  "akamai",
		EnableShellCompletion: true,
		HideHelp:              true,
		Writer:                outbuf,
		Commands: []*cli.Command{
			{
				Name:          "help",
				ShellComplete: Default,
			},
			{
				Name:          "test",
				ShellComplete: Default,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "flag",
						Aliases: []string{"f"},
					},
				},
				HideHelp: true,
			},
			{
				Name: "test2",
			},
		},
	}

	complete(t, app, "akamai", "help", "test", "--generate-shell-completion")

	assert.Equal(t, "--flag\n-f\n", outbuf.String())
}

func complete(t *testing.T, cmd *cli.Command, args ...string) {
	t.Helper()

	previousArgs := os.Args
	os.Args = args
	t.Cleanup(func() {
		os.Args = previousArgs
	})

	require.NoError(t, cmd.Run(context.Background(), args))
}
