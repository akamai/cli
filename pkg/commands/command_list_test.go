package commands

import (
	"fmt"
	"os"
	"testing"

	"github.com/akamai/cli/pkg/color"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCmdListWithRemote(t *testing.T) {
	tests := map[string]struct {
		args      []string
		init      func(*mocked)
		packages  *packageList
		withError string
	}{
		"list all commands": {
			init: func(m *mocked) {
				m.term.On("Writeln", []interface{}{color.YellowString("\nInstalled Commands:\n")}).Return(0, nil).Once()

				// List command
				m.term.On("Printf", color.BoldString("  list"), []interface{}(nil)).Return().Once()
				m.term.On("Printf", " (%s: ", []interface{}{"aliases"}).Return().Once()
				m.term.On("Printf", color.BoldString("ls"), []interface{}(nil)).Return().Once()
				m.term.On("Printf", ", ", []interface{}(nil)).Return().Once()
				m.term.On("Printf", color.BoldString("show"), []interface{}(nil)).Return().Once()
				m.term.On("Printf", ")", []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}(nil)).Return(0, nil).Once()
				m.term.On("Printf", "    Displays available commands\n", []interface{}(nil)).Return().Once()

				// Help command
				m.term.On("Printf", color.BoldString("  help"), []interface{}(nil)).Return().Once()
				m.term.On("Printf", " (%s: ", []interface{}{"alias"}).Return().Once()
				m.term.On("Printf", color.BoldString("h"), []interface{}(nil)).Return().Once()
				m.term.On("Printf", ")", []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}(nil)).Return(0, nil).Once()

				m.term.On("Printf", "\nSee \"%s\" for details.\n", []interface{}{color.BlueString("%s help [command]", tools.Self())}).Return().Once()

				// List --remote command
				m.term.On("Writeln", []interface{}{color.YellowString("\nAvailable Commands:\n\n")}).Return(0, nil).Once()
				m.term.On("Printf", color.BoldString("  ClI-1"), []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}{fmt.Sprintf(" [package: %s]", color.BlueString("abc-1"))}).Return(0, nil).Once()
				m.term.On("Printf", "    test for match on title\n", []interface{}(nil)).Return().Once()

				m.term.On("Printf", color.BoldString("  cli-2"), []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}{fmt.Sprintf(" [package: %s]", color.BlueString("cli-2"))}).Return(0, nil).Once()
				m.term.On("Printf", "    test for match on name\n", []interface{}(nil)).Return().Once()

				m.term.On("Printf", color.BoldString("  abc-3"), []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}{fmt.Sprintf(" [package: %s]", color.BlueString("abc-3"))}).Return(0, nil).Once()
				m.term.On("Printf", "    CLI - test for match on description\n", []interface{}(nil)).Return().Once()

				m.term.On("Printf", color.BoldString("  abc-4"), []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}{fmt.Sprintf(" [package: %s]", color.BlueString("abc-4"))}).Return(0, nil).Once()
				m.term.On("Printf", "    abc - no match\n", []interface{}(nil)).Return().Once()

				m.term.On("Printf", color.BoldString("  cli"), []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}{fmt.Sprintf(" [package: %s]", color.BlueString("abc-5"))}).Return(0, nil).Once()
				m.term.On("Printf", "    test for match on command name\n", []interface{}(nil)).Return().Once()

				m.term.On("Printf", color.BoldString("  abc-6"), []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}{fmt.Sprintf(" [package: %s]", color.BlueString("cli-no-cmd-match"))}).Return(0, nil).Once()
				m.term.On("Printf", "    title and name match, but no match on command\n", []interface{}(nil)).Return().Once()

				m.term.On("Printf", color.BoldString("  sample"), []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}{fmt.Sprintf(" [package: %s]", color.BlueString("SAMPLE"))}).Return(0, nil).Once()
				m.term.On("Printf", "    test for single match\n", []interface{}(nil)).Return().Once()

				m.term.On("Printf", color.BoldString("  echo-uninstall"), []interface{}(nil)).Return().Once()
				m.term.On("Writeln", []interface{}{fmt.Sprintf(" [package: %s]", color.BlueString("echo"))}).Return(0, nil).Once()
				m.term.On("Printf", "    test for single match\n", []interface{}(nil)).Return().Once()

				m.term.On("Printf", "\nInstall using \"%s\".\n", []interface{}{color.BlueString("%s install [package]", tools.Self())}).Return().Once()
			},
			packages: packagesForTest,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := &mocked{&terminal.Mock{}, &config.Mock{}, nil, nil, nil}
			pr := &mockPackageReader{}
			pr.On("readPackage").Return(test.packages.copy(t), nil).Once()

			commandToExecute := &cli.Command{
				Name: "list",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "remote",
					},
				},
				Description: "Displays available commands",
				Aliases:     []string{"ls", "show"},
				Action: func(context *cli.Context) error {
					return cmdListWithPackageReader(context, pr)
				},
			}

			app, ctx := setupTestApp(commandToExecute, m)
			args := os.Args[0:1]
			args = append(args, "list", "--remote")

			test.init(m)
			err := app.RunContext(ctx, args)

			m.cfg.AssertExpectations(t)
			m.term.AssertExpectations(t)
			if test.withError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.withError)
				return
			}
			require.NoError(t, err)
		})
	}
}
