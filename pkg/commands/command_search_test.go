package commands

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCmdSearch(t *testing.T) {
	tests := map[string]struct {
		args      []string
		init      func(*terminal.Mock)
		packages  *packageList
		withError string
	}{
		"search and find single package - sample": {
			args: []string{"sample"},
			init: func(m *terminal.Mock) {
				bold := color.New(color.FgWhite, color.Bold)
				m.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{1})

				m.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"sample", color.BlueString("SAMPLE")}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Command:")+" %s %s\n", []interface{}{"sample", ""}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Version:")+" %s\n", []interface{}{"2.0.0"}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Description:")+" %s\n\n", []interface{}{"test for single match"}).
					Return().Once()

				m.On("Printf", "\nInstall using \"%s\".\n", []interface{}{color.BlueString("%s install [package]", tools.Self())}).
					Return().Once()
			},
			packages: packagesForTest,
		},
		"search and find multiple packages - cli": {
			args: []string{"cli"},
			init: func(m *terminal.Mock) {
				bold := color.New(color.FgWhite, color.Bold)
				m.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{5})

				m.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"CLI no cmd match", color.BlueString("cli-no-cmd-match")}).
					Return().Once()

				m.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"abc-2", color.BlueString("cli-2")}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Command:")+" %s %s\n", []interface{}{"cli-2", "(aliases: abc, abc2)"}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Description:")+" %s\n\n", []interface{}{"test for match on name"}).
					Return().Once()

				m.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"cli-1", color.BlueString("abc-1")}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Command:")+" %s %s\n", []interface{}{"ClI-1", ""}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Description:")+" %s\n\n", []interface{}{"test for match on title"}).
					Return().Once()

				m.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"abc-5", color.BlueString("abc-5")}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Command:")+" %s %s\n", []interface{}{"cli", ""}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Description:")+" %s\n\n", []interface{}{"test for match on command name"}).
					Return().Once()

				m.On("Printf", color.GreenString("Package: ")+"%s [%s]\n", []interface{}{"abc-3", color.BlueString("abc-3")}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Command:")+" %s %s\n", []interface{}{"abc-3", ""}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Version:")+" %s\n", []interface{}{"1.0.0"}).
					Return().Once()
				m.On("Printf", bold.Sprintf("  Description:")+" %s\n\n", []interface{}{"CLI - test for match on description"}).
					Return().Once()

				m.On("Printf", "\nInstall using \"%s\".\n", []interface{}{color.BlueString("%s install [package]", tools.Self())}).
					Return().Once()
			},
			packages: packagesForTest,
		},
		"search with no results - terraform": {
			args: []string{"terraform"},
			init: func(m *terminal.Mock) {
				m.On("Printf", color.YellowString("Results Found:")+" %d\n\n", []interface{}{0})
			},
			packages: packagesForTest,
		},
		"no args passed": {
			args:      []string{},
			init:      func(m *terminal.Mock) {},
			withError: "You must specify one or more keywords",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := &mocked{&terminal.Mock{}, &config.Mock{}, nil, nil, nil}
			pr := &mockPackageReader{}
			pr.On("readPackage").Return(test.packages.copy(t), nil).Once()

			commandToExecute := &cli.Command{
				Name: "search",
				Action: func(context *cli.Context) error {
					return cmdSearchWithPackageReader(context, pr)
				},
			}

			app, ctx := setupTestApp(commandToExecute, m)
			args := os.Args[0:1]
			args = append(args, "search")
			args = append(args, test.args...)

			test.init(m.term)
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

// copy returns copy of packageList content
func (p *packageList) copy(t *testing.T) *packageList {
	bytes, err := json.Marshal(p)
	require.NoError(t, err)
	var pl packageList
	err = json.Unmarshal(bytes, &pl)
	require.NoError(t, err)

	return &pl
}

// packagesForTest is a package list with example packages
var packagesForTest = &packageList{
	Version: 1.0,
	Packages: []packageListItem{
		{
			Title: "cli-1",
			Name:  "abc-1",
			Commands: []command{
				{
					Name:        "ClI-1",
					Version:     "1.0.0",
					Description: "test for match on title",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title:   "abc-2",
			Name:    "cli-2",
			Version: "1.0.0",
			Commands: []command{
				{
					Name:        "cli-2",
					Aliases:     []string{"abc", "abc2"},
					Version:     "1.0.0",
					Description: "test for match on name",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title: "abc-3",
			Name:  "abc-3",
			Commands: []command{
				{
					Name:        "abc-3",
					Version:     "1.0.0",
					Description: "CLI - test for match on description",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title: "abc-4",
			Name:  "abc-4",
			Commands: []command{
				{
					Name:        "abc-4",
					Version:     "1.0.0",
					Description: "abc - no match",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title: "abc-5",
			Name:  "abc-5",
			Commands: []command{
				{
					Name:        "cli",
					Version:     "1.0.0",
					Description: "test for match on command name",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title: "CLI no cmd match",
			Name:  "cli-no-cmd-match",
			Commands: []command{
				{
					Name:        "abc-6",
					Version:     "1.0.0",
					Description: "title and name match, but no match on command",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
		{
			Title: "sample",
			Name:  "SAMPLE",
			Commands: []command{
				{
					Name:        "sample",
					Version:     "2.0.0",
					Description: "test for single match",
				},
			},
			Requirements: requirements{
				Node: "7.0.0",
			},
		},
	},
}
