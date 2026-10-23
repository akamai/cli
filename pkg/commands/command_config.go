package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/config"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/urfave/cli/v3"
)

func cmdConfigSet(ctx context.Context, cmd *cli.Command) (e error) {
	ctx = log.WithCommandContext(ctx, cmd.Name)
	logger := log.FromContext(ctx)
	start := time.Now()
	logger.Debug("CONFIG SET START")
	defer func() {
		if e == nil {
			logger.Debug(fmt.Sprintf("CONFIG SET FINISH: %v", time.Since(start)))
		} else {
			logger.Error(fmt.Sprintf("CONFIG SET ERROR: %v", e))
		}
	}()
	cfg := config.Get(ctx)

	section, key, err := parseConfigPath(cmd)
	if err != nil {
		logger.Error(fmt.Sprintf("Error parsing config path: %v", err))
		return cli.Exit(color.RedString("Unable to set config value: %v", err), 1)
	}

	value := strings.Join(cmd.Args().Tail(), " ")
	cfg.SetValue(section, key, value)
	if err := cfg.Save(ctx); err != nil {
		logger.Error(fmt.Sprintf("Error saving config: %v", err))
		return cli.Exit(color.RedString("Unable to set config value: %v", err), 1)
	}

	return nil
}

func cmdConfigGet(ctx context.Context, cmd *cli.Command) (e error) {
	ctx = log.WithCommandContext(ctx, cmd.Name)
	logger := log.FromContext(ctx)
	start := time.Now()
	logger.Debug("CONFIG GET START")
	defer func() {
		if e == nil {
			logger.Debug(fmt.Sprintf("CONFIG GET FINISH: %v", time.Since(start)))
		} else {
			logger.Error(fmt.Sprintf("CONFIG GET ERROR: %v", e))
		}
	}()
	cfg := config.Get(ctx)

	section, key, err := parseConfigPath(cmd)
	if err != nil {
		logger.Error(fmt.Sprintf("Error parsing config path: %v", err))
		return cli.Exit(color.RedString("Unable to get config value: %v", err), 1)
	}

	val, _ := cfg.GetValue(section, key)
	if _, err := terminal.Get(ctx).Writeln(val); err != nil {
		return err
	}
	logger.Debug(val)

	return nil
}

func cmdConfigUnset(ctx context.Context, cmd *cli.Command) (e error) {
	ctx = log.WithCommandContext(ctx, cmd.Name)
	logger := log.FromContext(ctx)
	start := time.Now()
	logger.Debug("CONFIG UNSET START")
	defer func() {
		if e == nil {
			logger.Debug(fmt.Sprintf("CONFIG UNSET FINISH: %v", time.Since(start)))
		} else {
			logger.Error(fmt.Sprintf("CONFIG UNSET ERROR: %v", e))
		}
	}()
	cfg := config.Get(ctx)
	section, key, err := parseConfigPath(cmd)
	if err != nil {
		logger.Error(fmt.Sprintf("Error parsing config path: %v", err))
		return cli.Exit(color.RedString("Unable to unset config value: %v", err), 1)
	}

	cfg.UnsetValue(section, key)
	if err := cfg.Save(ctx); err != nil {
		logger.Error(fmt.Sprintf("Error saving config: %v", err))
		return cli.Exit(color.RedString("Unable to set config value: %v", err), 1)
	}
	return nil
}

func cmdConfigList(ctx context.Context, cmd *cli.Command) (e error) {
	ctx = log.WithCommandContext(ctx, cmd.Name)
	logger := log.FromContext(ctx)
	start := time.Now()
	logger.Debug("CONFIG LIST START")
	defer func() {
		if e == nil {
			logger.Debug(fmt.Sprintf("CONFIG LIST FINISH: %v", time.Since(start)))
		} else {
			logger.Error(fmt.Sprintf("CONFIG LIST ERROR: %v", e))
		}
	}()
	cfg := config.Get(ctx)
	term := terminal.Get(ctx)

	allValues := cfg.Values()
	if cmd.Args().Present() {
		sectionName := cmd.Args().First()
		section, ok := allValues[sectionName]
		if !ok {
			return nil
		}
		for key, value := range section {
			term.Printf("%s.%s = %s\n", sectionName, key, value)
		}

		return nil
	}

	for sectionName, section := range allValues {
		for key, value := range section {
			term.Printf("%s.%s = %s\n", sectionName, key, value)
		}
	}
	return nil
}

func parseConfigPath(cmd *cli.Command) (string, string, error) {
	path := strings.Split(cmd.Args().First(), ".")
	if len(path) < 2 {
		return "", "", fmt.Errorf("section key has to be provided in <section>.<key> format")
	}
	section := path[0]
	key := strings.Join(path[1:], "-")
	return section, key, nil
}
