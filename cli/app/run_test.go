package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestFindCollisions(t *testing.T) {
	tests := map[string]struct {
		availableCmds []*cli.Command
		args          []string
		withError     string
	}{
		"no command": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall"},
				},
			},
			args: []string{"akamai"},
		},
		"no collision": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall"},
				},
			},
			args: []string{"akamai", "echo"},
		},
		"Collision on name, but not the requested command": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall2/firewall"},
				},
			},
			args: []string{"akamai", "echo"},
		},
		"Collision on command name, but not with package prefix": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall2/firewall"},
				},
			},
			args: []string{"akamai", "firewall/firewall"},
		},
		"Collision on fully qualified command name": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
			},
			args:      []string{"akamai", "firewall/firewall"},
			withError: `this command is ambiguous`,
		},
		"Collision on command name": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"fw", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"fw2", "firewall2/firewall"},
				},
			},
			args:      []string{"akamai", "firewall"},
			withError: `this command is ambiguous`,
		},
		"Help command: no collision": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall"},
				},
			},
			args: []string{"akamai", "help", "echo"},
		},
		"Help command: collision on name, but not the requested command": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall2/firewall"},
				},
			},
			args: []string{"akamai", "help", "echo"},
		},
		"Help command: collision on command name, but not with package prefix": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall2/firewall"},
				},
			},
			args: []string{"akamai", "help", "firewall/firewall"},
		},
		"Help command: collision on fully qualified command name": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
			},
			args:      []string{"akamai", "help", "firewall/firewall"},
			withError: `this command is ambiguous`,
		},
		"Help command: collision on command name": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"fw", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"fw2", "firewall2/firewall"},
				},
			},
			args:      []string{"akamai", "help", "firewall"},
			withError: `this command is ambiguous`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := findCollisions(test.availableCmds, test.args)
			if test.withError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.withError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestFindDuplicate(t *testing.T) {
	tests := map[string]struct {
		availableCmds []*cli.Command
		cmdName       string
		withError     string
	}{
		"no duplicates": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall"},
				},
			},
			cmdName: "echo",
		},
		"Duplicate name, but not the requested command": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall2/firewall"},
				},
			},
			cmdName: "echo",
		},
		"Duplicate command name, but not with package prefix": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall2/firewall"},
				},
			},
			cmdName: "firewall/firewall",
		},
		"Duplicated fully qualified command name": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"firewall", "firewall/firewall"},
				},
			},
			cmdName:   "firewall/firewall",
			withError: `this command is ambiguous`,
		},
		"Duplicate command name": {
			availableCmds: []*cli.Command{
				{
					Name: "list",
				},
				{
					Name: "help",
				},
				{
					Name: "uninstall",
				},
				{
					Name:    "install",
					Aliases: []string{"get"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"fw", "firewall/firewall"},
				},
				{
					Name:    "firewall",
					Aliases: []string{"fw2", "firewall2/firewall"},
				},
			},
			cmdName:   "firewall",
			withError: `this command is ambiguous`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := findDuplicate(test.availableCmds, test.cmdName)
			if test.withError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.withError)
				return
			}
			require.NoError(t, err)
		})
	}
}
