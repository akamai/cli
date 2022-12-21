package autocomplete

import (
	"bytes"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestComplete_root_cmd(t *testing.T) {
	outbuf := &bytes.Buffer{}
	app := cli.NewApp()
	app.Commands = []*cli.Command{
		{
			Name: "test",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name: "flag",
				},
			},
		},
		{
			Name: "test2",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name: "flag2",
				},
			},
		},
	}
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name: "gflag1",
		},
		&cli.StringFlag{
			Name: "gflag2",
		},
	}
	app.Writer = outbuf

	fs := flag.NewFlagSet("test", flag.PanicOnError)
	ctx := cli.NewContext(app, fs, nil)

	Default(ctx)

	assert.Equal(t, "test\ntest2\n--gflag1\n--gflag2\n", outbuf.String())
}

func TestComplete_subcommand(t *testing.T) {
	outbuf := &bytes.Buffer{}
	app := cli.NewApp()
	app.Writer = outbuf

	fs := flag.NewFlagSet("test", flag.PanicOnError)
	ctx := cli.NewContext(app, fs, nil)
	ctx.Command = &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "flag",
			},
		},
		Subcommands: []*cli.Command{
			{
				Name: "sub1",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "sub1flag1",
					},
					&cli.StringFlag{
						Name: "sub1flag2",
					},
				},
			},
			{
				Name: "sub2",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "sub2flag1",
					},
					&cli.StringFlag{
						Name: "sub2flag2",
					},
				},
				Hidden: true,
			},
		},
	}

	Default(ctx)

	assert.Equal(t, "sub1\n--flag\n", outbuf.String())
}

func TestComplete_short_flag(t *testing.T) {
	outbuf := &bytes.Buffer{}
	app := cli.NewApp()
	app.Writer = outbuf

	fs := flag.NewFlagSet("test", flag.PanicOnError)
	ctx := cli.NewContext(app, fs, nil)
	ctx.Command = &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "flag",
				Aliases: []string{"f"},
			},
		},
	}

	Default(ctx)

	assert.Equal(t, "--flag\n-f\n", outbuf.String())
}

func TestComplete_help_command(t *testing.T) {
	outbuf := &bytes.Buffer{}
	app := cli.NewApp()
	app.Writer = outbuf
	app.EnableBashCompletion = true
	app.BashComplete = Default

	app.Commands = []*cli.Command{
		{
			Name:         "help",
			BashComplete: Default,
		},
		{
			Name: "test",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "flag",
					Aliases: []string{"f"},
				},
			},
			HideHelp:     true,
			BashComplete: Default,
		},
		{
			Name: "test2",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name: "flag2",
				},
			},
		},
	}

	app.Run([]string{"akamai", "help", "test", "--generate-bash-completion"})

	assert.Equal(t, "--flag\n-f\n", outbuf.String())
}
