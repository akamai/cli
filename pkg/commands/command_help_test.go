package commands

import (
	"bytes"
	"os"
	"regexp"
	"testing"

	"github.com/akamai/cli/pkg/app"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCmdHelp(t *testing.T) {
	tests := map[string]struct {
		args           []string
		cmd            *cli.Command
		expectedOutput *regexp.Regexp
		withError      string
	}{
		"full help": {
			args: []string{},
			cmd: &cli.Command{
				Name:        "test",
				Description: "test command",
				Category:    "",
			},
			expectedOutput: regexp.MustCompile(`.*Usage: \n.*command \[command flags] \[arguments...]\n\n.*Built-In Commands:\n.*test.*\n.*help.*`),
		},
		"help for specific command": {
			args: []string{"test"},
			cmd: &cli.Command{
				Name:        "test",
				Description: "test command",
				Category:    "",
			},
			expectedOutput: regexp.MustCompile(`.*Name: \n.*test\n\n.*Usage: \n.*test \[arguments...]\n\n.*Description: \n.*test command\n\n`),
		},
		"help for installed command": {
			args: []string{"test"},
			cmd: &cli.Command{
				Name:        "test",
				Description: "test command",
				Category:    "Installed command",
			},
			expectedOutput: regexp.MustCompile(`.*Name: \n.*help\n\n.*Usage: \n.*help \[command] \[sub-command]\n\n.*Description: \n.*Displays help information\n\n`),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := &mocked{&terminal.Mock{}, &config.Mock{}, nil, nil}
			wr := bytes.Buffer{}
			testApp, ctx := setupTestApp(test.cmd, m)
			testApp.Commands = append(testApp.Commands, &cli.Command{
				Name:         "help",
				ArgsUsage:    "[command] [sub-command]",
				Description:  "Displays help information",
				Action:       cmdHelp,
				HideHelp:     true,
				BashComplete: app.DefaultAutoComplete,
			})
			app.SetHelpTemplates()
			testApp.Writer = &wr
			args := os.Args[0:1]
			args = append(args, "help")
			args = append(args, test.args...)

			err := testApp.RunContext(ctx, args)
			if test.withError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.withError)
				return
			}
			assert.Regexp(t, test.expectedOutput, wr.String())
			require.NoError(t, err)
		})
	}
}
