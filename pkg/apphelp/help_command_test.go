package apphelp

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"

	"os"
	"testing"

	"github.com/akamai/cli/v2/pkg/config"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCmdHelp(t *testing.T) {
	var binarySuffix string
	if runtime.GOOS == "windows" {
		binarySuffix = ".exe"
	}
	tests := map[string]struct {
		args           []string
		cmd            *cli.Command
		expectedOutput string
	}{
		"app help": {
			args: []string{},
			cmd: &cli.Command{
				Name:        "test",
				Description: "test command",
				Category:    "",
			},
			expectedOutput: fmt.Sprintf(`
Usage:
  apphelp.test%s [global flags] command [command flags] [arguments...]

Commands:
  help
  test

Global Flags:
  --edgerc value, -e value   edgerc config path passed to executed commands, defaults to ~/.edgerc
  --section value, -s value  edgerc section name passed to executed commands, defaults to 'default'
  --help, -h                 show help (default: false)`, binarySuffix),
		},

		"help for specific command": {
			args: []string{"test"},
			cmd: &cli.Command{
				Name:        "test",
				Description: "test command",
				Category:    "",
				ArgsUsage:   "<arg1> <arg2>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "test-flag",
						Usage: "this is a test flag",
					},
				},
			},
			expectedOutput: fmt.Sprintf(`
Name:
  apphelp.test%s test

Usage:
  apphelp.test%s [global flags] test [command flags] <arg1> <arg2>

Description:
  test command

Command Flags:
  --test-flag  this is a test flag (default: false)
  --help, -h   show help (default: false)

Global Flags:
  --edgerc value, -e value   edgerc config path passed to executed commands, defaults to ~/.edgerc
  --section value, -s value  edgerc section name passed to executed commands, defaults to 'default'
`, binarySuffix, binarySuffix),
		},
		"help for specific command alias": {
			args: []string{"test-alias"},
			cmd: &cli.Command{
				Name:        "test",
				Aliases:     []string{"test-alias"},
				Description: "test command",
				Category:    "",
				ArgsUsage:   "<arg1> <arg2>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "test-flag",
						Usage: "this is a test flag",
					},
				},
			},
			expectedOutput: fmt.Sprintf(`
Name:
  apphelp.test%s test

Usage:
  apphelp.test%s [global flags] test [command flags] <arg1> <arg2>

Description:
  test command

Command Flags:
  --test-flag  this is a test flag (default: false)
  --help, -h   show help (default: false)

Global Flags:
  --edgerc value, -e value   edgerc config path passed to executed commands, defaults to ~/.edgerc
  --section value, -s value  edgerc section name passed to executed commands, defaults to 'default'
`, binarySuffix, binarySuffix),
		},

		"help for a subcommand": {
			args: []string{"test", "subcommand"},
			cmd: &cli.Command{
				Name:        "test",
				Description: "test command",
				Category:    "",
				Action:      nil,
				Subcommands: []*cli.Command{
					{
						Name:        "subcommand",
						Description: "a test subcommand without a category",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "test-flag",
								Usage: "this is a test flag",
							},
						},
					},
				},
			},
			expectedOutput: fmt.Sprintf(`
Name:
  apphelp.test%s test subcommand

Usage:
  apphelp.test%s [global flags] test subcommand [command flags] 

Description:
  a test subcommand without a category

Command Flags:
  --test-flag value  this is a test flag
  --help, -h         show help (default: false)

Global Flags:
  --edgerc value, -e value   edgerc config path passed to executed commands, defaults to ~/.edgerc
  --section value, -s value  edgerc section name passed to executed commands, defaults to 'default'
`, binarySuffix, binarySuffix),
		},

		"help for command with subcommands": {
			args: []string{"test"},
			cmd: &cli.Command{
				Name:        "test",
				Description: "test command",
				Category:    "",
				Action:      func(_ *cli.Context) error { fmt.Println("oops!"); return nil },
				Subcommands: []*cli.Command{
					{
						Name:        "subcommand-no-category",
						Description: "a test subcommand without a category",
					},
					{
						Name:        "subcommand-with-aliases",
						Description: "a test subcommand with aliases and without a category",
						Aliases:     []string{"sub-wa", "s-w-a"},
					},
					{
						Name:        "subcommand-in-category1",
						Description: "a test subcommand in category 1",
						Category:    "category1",
					},
					{
						Name:        "subcommand-in-category2",
						Description: "a test subcommand in category 2",
						Category:    "category2",
					},
				},
			},
			expectedOutput: fmt.Sprintf(`
Name:
  apphelp.test%s test

Usage:
  apphelp.test%s [global flags] test [command flags] <subcommand>

Description:
  test command

Subcommands:
  subcommand-no-category
  subcommand-with-aliases (aliases: sub-wa, s-w-a)
  help (alias: h)
category1:
  subcommand-in-category1
category2:
  subcommand-in-category2

Command Flags:
  --help, -h  show help (default: false)

Global Flags:
  --edgerc value, -e value   edgerc config path passed to executed commands, defaults to ~/.edgerc
  --section value, -s value  edgerc section name passed to executed commands, defaults to 'default'
`, binarySuffix, binarySuffix),
		},

		"help for command with optional subcommands": {
			args: []string{"test"},
			cmd: &cli.Command{
				Name:        "test",
				Description: "test command",
				Category:    "",
				ArgsUsage:   "<usage>",
				Action:      func(_ *cli.Context) error { fmt.Println("oops!"); return nil },
				Subcommands: []*cli.Command{
					{
						Name:        "subcommand-no-category",
						Description: "a test subcommand without a category",
					},
					{
						Name:        "subcommand-with-aliases",
						Description: "a test subcommand with aliases and without a category",
						Aliases:     []string{"sub-wa", "s-w-a"},
					},
					{
						Name:        "subcommand-in-category1",
						Description: "a test subcommand in category 1",
						Category:    "category1",
					},
					{
						Name:        "subcommand-in-category2",
						Description: "a test subcommand in category 2",
						Category:    "category2",
					},
				},
			},
			expectedOutput: fmt.Sprintf(`
Name:
  apphelp.test%s test

Usage:
  apphelp.test%s [global flags] test [command flags] <usage>
  apphelp.test%s [global flags] test [command flags] <subcommand>

Description:
  test command

Subcommands:
  subcommand-no-category
  subcommand-with-aliases (aliases: sub-wa, s-w-a)
  help (alias: h)
category1:
  subcommand-in-category1
category2:
  subcommand-in-category2

Command Flags:
  --help, -h  show help (default: false)

Global Flags:
  --edgerc value, -e value   edgerc config path passed to executed commands, defaults to ~/.edgerc
  --section value, -s value  edgerc section name passed to executed commands, defaults to 'default'
`, binarySuffix, binarySuffix, binarySuffix),
		},

		"help for subcommand no category": {
			args: []string{"test", "subcommand-no-category"},
			cmd: &cli.Command{
				Name:        "test",
				Description: "test command",
				Category:    "",
				Action:      func(_ *cli.Context) error { fmt.Println("oops!"); return nil },
				Subcommands: []*cli.Command{
					{
						Name:        "subcommand-no-category",
						Description: "a test subcommand without a category",
					},
					{
						Name:        "subcommand-with-aliases",
						Description: "a test subcommand with aliases and without a category",
						Aliases:     []string{"sub-wa", "s-w-a"},
					},
					{
						Name:        "subcommand-in-category1",
						Description: "a test subcommand in category 1",
						Category:    "category1",
					},
					{
						Name:        "subcommand-in-category2",
						Description: "a test subcommand in category 2",
						Category:    "category2",
					},
				},
			},
			expectedOutput: fmt.Sprintf(`
Name:
  apphelp.test%s test subcommand-no-category

Usage:
  apphelp.test%s [global flags] test subcommand-no-category [command flags] 

Description:
  a test subcommand without a category

Command Flags:
  --help, -h  show help (default: false)

Global Flags:
  --edgerc value, -e value   edgerc config path passed to executed commands, defaults to ~/.edgerc
  --section value, -s value  edgerc section name passed to executed commands, defaults to 'default'
`, binarySuffix, binarySuffix),
		},

		"help for subcommands with category": {
			args: []string{"test", "subcommand-in-category1"},
			cmd: &cli.Command{
				Name:        "test",
				Description: "test command",
				Category:    "",
				Action:      func(_ *cli.Context) error { fmt.Println("oops!"); return nil },
				Subcommands: []*cli.Command{
					{
						Name:        "subcommand-no-category",
						Description: "a test subcommand without a category",
					},
					{
						Name:        "subcommand-with-aliases",
						Description: "a test subcommand with aliases and without a category",
						Aliases:     []string{"sub-wa", "s-w-a"},
					},
					{
						Name:        "subcommand-in-category1",
						Description: "a test subcommand in category 1",
						Category:    "category1",
					},
					{
						Name:        "subcommand-in-category2",
						Description: "a test subcommand in category 2",
						Category:    "category2",
					},
				},
			},
			expectedOutput: fmt.Sprintf(`
Name:
  apphelp.test%s test subcommand-in-category1

Usage:
  apphelp.test%s [global flags] test subcommand-in-category1 [command flags] 

Type:
  category1

Description:
  a test subcommand in category 1

Command Flags:
  --help, -h  show help (default: false)

Global Flags:
  --edgerc value, -e value   edgerc config path passed to executed commands, defaults to ~/.edgerc
  --section value, -s value  edgerc section name passed to executed commands, defaults to 'default'
`, binarySuffix, binarySuffix),
		},

		"help for command with simplified template": {
			args: []string{"test"},
			cmd: &cli.Command{
				Name:        "test",
				Description: "test command",
				Category:    "",
				ArgsUsage:   "<arg1> <arg2>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "test-flag",
						Usage: "this is a test flag",
					},
				},
				CustomHelpTemplate: SimplifiedHelpTemplate,
			},
			expectedOutput: fmt.Sprintf(`
Name:
  apphelp.test%s test

Usage:
  apphelp.test%s test [command flags]

Description:
  test command

Command Flags:
  --test-flag  this is a test flag (default: false)
  --help, -h   show help (default: false)
`, binarySuffix, binarySuffix),
		},
	}

	globalFlags := []cli.Flag{
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

	for name, test := range tests {
		appArgs := map[string][]string{
			"command": append([]string{os.Args[0], "help"}, test.args...),
			"flag":    append([]string{os.Args[0]}, append(test.args, "--help")...),
		}
		for helpType, args := range appArgs {
			t.Run(fmt.Sprintf("%s - %s", name, helpType), func(t *testing.T) {
				wr := bytes.Buffer{}
				ctx := terminal.Context(context.Background(), &terminal.Mock{})
				ctx = config.Context(ctx, &config.Mock{})

				testApp := cli.NewApp()
				testApp.Writer = &wr
				testApp.Flags = globalFlags
				Setup(testApp)
				testApp.Commands = append(testApp.Commands, test.cmd)

				err := testApp.RunContext(ctx, args)
				require.NoError(t, err)
				assert.Equal(t, strings.TrimPrefix(test.expectedOutput, "\n"), wr.String())
			})
		}
	}
}
