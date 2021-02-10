package commands

import (
	"context"
	"fmt"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"os"
	"testing"
)

type mocked struct {
	term *terminal.Mock
	cfg  *config.Mock
}

func TestCmdConfigSet(t *testing.T) {
	tests := map[string]struct {
		args      []string
		init      func(*config.Mock)
		withError string
	}{
		"set config no error": {
			args: []string{"cli.testKey", "testValue"},
			init: func(m *config.Mock) {
				m.On("SetValue", "cli", "testKey", "testValue").Return().Once()
				m.On("Save").Return(nil).Once()
			},
		},
		"key format error": {
			args:      []string{"cli", "testKey", "testValue"},
			init:      func(m *config.Mock) {},
			withError: "Unable to set config value: section key has to be provided in <section>.<key> format",
		},
		"error on save": {
			args: []string{"cli.testKey", "testValue"},
			init: func(m *config.Mock) {
				m.On("SetValue", "cli", "testKey", "testValue").Return().Once()
				m.On("Save").Return(fmt.Errorf("save error")).Once()
			},
			withError: "save error",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := &mocked{&terminal.Mock{}, &config.Mock{}}
			command := &cli.Command{
				Name: "config",
				Subcommands: []*cli.Command{
					{
						Name:   "set",
						Action: cmdConfigSet,
					},
				},
			}
			app, ctx := setupTestApp(command, m)
			args := os.Args[0:1]
			args = append(args, "config", "set")
			args = append(args, test.args...)

			test.init(m.cfg)
			err := app.RunContext(ctx, args)

			m.cfg.AssertExpectations(t)
			if test.withError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.withError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCmdConfigGet(t *testing.T) {
	tests := map[string]struct {
		args      []string
		init      func(*mocked)
		withError string
	}{
		"get config value no error": {
			args: []string{"cli.testKey"},
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "testKey").Return("test val", true).Once()

				m.term.On("Writeln", []interface{}{"test val"}).Return(0, nil).Once()
			},
		},
		"key format error": {
			args:      []string{"cli"},
			init:      func(m *mocked) {},
			withError: "Unable to get config value: section key has to be provided in <section>.<key> format",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := &mocked{&terminal.Mock{}, &config.Mock{}}
			command := &cli.Command{
				Name: "config",
				Subcommands: []*cli.Command{
					{
						Name:   "get",
						Action: cmdConfigGet,
					},
				},
			}
			app, ctx := setupTestApp(command, m)
			args := os.Args[0:1]
			args = append(args, "config", "get")
			args = append(args, test.args...)

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

func TestCmdConfigUnset(t *testing.T) {
	tests := map[string]struct {
		args      []string
		init      func(*config.Mock)
		withError string
	}{
		"unset config value no error": {
			args: []string{"cli.testKey", "testValue"},
			init: func(m *config.Mock) {
				m.On("UnsetValue", "cli", "testKey").Return().Once()
				m.On("Save").Return(nil).Once()
			},
		},
		"key format error": {
			args:      []string{"cli", "testKey"},
			init:      func(m *config.Mock) {},
			withError: "Unable to unset config value: section key has to be provided in <section>.<key> format",
		},
		"error on save": {
			args: []string{"cli.testKey", "testValue"},
			init: func(m *config.Mock) {
				m.On("UnsetValue", "cli", "testKey").Return().Once()
				m.On("Save").Return(fmt.Errorf("save error")).Once()
			},
			withError: "save error",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := &mocked{&terminal.Mock{}, &config.Mock{}}
			command := &cli.Command{
				Name: "config",
				Subcommands: []*cli.Command{
					{
						Name:   "unset",
						Action: cmdConfigUnset,
					},
				},
			}
			app, ctx := setupTestApp(command, m)
			args := os.Args[0:1]
			args = append(args, "config", "unset")
			args = append(args, test.args...)

			test.init(m.cfg)
			err := app.RunContext(ctx, args)

			m.cfg.AssertExpectations(t)
			if test.withError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.withError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCmdConfigList(t *testing.T) {
	tests := map[string]struct {
		args      []string
		init      func(*mocked)
		withError string
	}{
		"list full config": {
			args: []string{},
			init: func(m *mocked) {
				m.cfg.On("Values").Return(map[string]map[string]string{
					"cli":  {"key1": "val1", "key2": "val2"},
					"test": {"key3": "val3"},
				}).Once()
				m.term.On("Printf", "%s.%s = %s\n", []interface{}{"cli", "key1", "val1"}).Return().Once()
				m.term.On("Printf", "%s.%s = %s\n", []interface{}{"cli", "key2", "val2"}).Return().Once()
				m.term.On("Printf", "%s.%s = %s\n", []interface{}{"test", "key3", "val3"}).Return().Once()
			},
		},
		"list specific section": {
			args: []string{"test"},
			init: func(m *mocked) {
				m.cfg.On("Values").Return(map[string]map[string]string{
					"cli":  {"key1": "val1", "key2": "val2"},
					"test": {"key3": "val3"},
				}).Once()
				m.term.On("Printf", "%s.%s = %s\n", []interface{}{"test", "key3", "val3"}).Return().Once()
			},
		},
		"section does not exist": {
			args: []string{"empty"},
			init: func(m *mocked) {
				m.cfg.On("Values").Return(map[string]map[string]string{
					"cli":  {"key1": "val1", "key2": "val2"},
					"test": {"key3": "val3"},
				}).Once()
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := &mocked{&terminal.Mock{}, &config.Mock{}}
			command := &cli.Command{
				Name: "config",
				Subcommands: []*cli.Command{
					{
						Name:   "list",
						Action: cmdConfigList,
					},
				},
			}
			app, ctx := setupTestApp(command, m)
			args := os.Args[0:1]
			args = append(args, "config", "list")
			args = append(args, test.args...)

			test.init(m)
			err := app.RunContext(ctx, args)

			m.cfg.AssertExpectations(t)
			if test.withError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.withError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func setupTestApp(command *cli.Command, m *mocked) (*cli.App, context.Context) {
	cli.OsExiter = func(rc int) {}
	ctx := terminal.Context(context.Background(), m.term)
	ctx = config.Context(ctx, m.cfg)
	app := cli.NewApp()
	app.Commands = []*cli.Command{command}
	return app, ctx
}
