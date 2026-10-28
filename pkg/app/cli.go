package app

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/akamai/cli/v2/pkg/apphelp"
	"github.com/akamai/cli/v2/pkg/autocomplete"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/tools"
	"github.com/akamai/cli/v2/pkg/version"

	"github.com/kardianos/osext"
	"github.com/urfave/cli/v3"
)

const sleep24HDuration = time.Hour * 24

// CreateApp creates and sets up *cli.Command
func CreateApp(ctx context.Context) *cli.Command {
	app := createAppTemplate(ctx, "", "Akamai CLI", "", version.Version, false)
	app.Flags = append(app.Flags,
		&cli.BoolFlag{
			Name:  "bash",
			Usage: "Output bash auto-complete",
		},
		&cli.BoolFlag{
			Name:  "zsh",
			Usage: "Output zsh auto-complete",
		},
		&cli.StringFlag{
			Name:  "proxy",
			Usage: "Set a proxy to use",
		},
		&cli.BoolFlag{
			Name:    "daemon",
			Usage:   "Keep Akamai CLI running in the background, particularly useful for Docker containers",
			Hidden:  true,
			Sources: cli.EnvVars("AKAMAI_CLI_DAEMON"),
		},
	)

	app.Action = func(ctx context.Context, cmd *cli.Command) error {
		return defaultAction(ctx, cmd)
	}

	app.Before = func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
		if cmd.IsSet("proxy") {
			proxy := cmd.String("proxy")
			if !strings.HasPrefix(proxy, "http://") && !strings.HasPrefix(proxy, "https://") {
				proxy = fmt.Sprintf("http://%s", proxy)
			}
			if err := os.Setenv("HTTP_PROXY", proxy); err != nil {
				return ctx, err
			}
			if err := os.Setenv("HTTPS_PROXY", proxy); err != nil {
				return ctx, err
			}
		}

		if cmd.IsSet("daemon") {
			for {
				time.Sleep(sleep24HDuration)
			}
		}
		return ctx, nil
	}

	return app
}

// CreateAppTemplate creates a basic *cli.Command template
func CreateAppTemplate(ctx context.Context, commandName, usage, description, version string) *cli.Command {
	return createAppTemplate(ctx, commandName, usage, description, version, true)
}

func createAppTemplate(ctx context.Context, commandName, usage, description, version string, useDefaults bool) *cli.Command {
	_, inCli := os.LookupEnv("AKAMAI_CLI")
	term := terminal.Get(ctx)

	appName := "akamai"
	if commandName != "" {
		appName = "akamai-" + commandName
		if inCli {
			appName = "akamai " + commandName
		}
	}

	app := &cli.Command{}
	app.Name = appName
	app.Usage = usage
	app.Description = description
	app.Version = version

	app.Copyright = "Copyright (C) Akamai Technologies, Inc"
	app.Writer = term
	app.ErrWriter = term.Error()
	app.EnableShellCompletion = true
	app.ShellComplete = autocomplete.Default

	var edgercpath, section string
	if useDefaults {
		edgercpath, _ = os.UserHomeDir()
		edgercpath = path.Join(edgercpath, ".edgerc")

		section = "default"
	}

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "edgerc",
			Aliases: []string{"e"},
			Usage:   "Location of the credentials file",
			Value:   edgercpath,
			Sources: cli.EnvVars("AKAMAI_EDGERC"),
		},
		&cli.StringFlag{
			Name:    "section",
			Aliases: []string{"s"},
			Usage:   "Section of the credentials file",
			Value:   section,
			Sources: cli.EnvVars("AKAMAI_EDGERC_SECTION"),
		},
		&cli.StringFlag{
			Name:    "accountkey",
			Aliases: []string{"account-key"},
			Usage:   "Account switch key",
			Sources: cli.EnvVars("AKAMAI_EDGERC_ACCOUNT_KEY"),
		},
	}

	cli.VersionFlag = &cli.BoolFlag{
		Name:  "version",
		Usage: "Output CLI version",
	}
	cli.HelpFlag = &cli.BoolFlag{
		Name:    "help",
		Aliases: []string{"h"},
		Usage:   "show help",
	}

	apphelp.Setup(app)

	return app
}

func defaultAction(ctx context.Context, c *cli.Command) error {
	cmd, err := osext.Executable()
	if err != nil {
		cmd = tools.Self()
	}

	zshScript := `set -k
# To enable zsh auto-completion, run: eval "$(` + cmd + ` --zsh)"
# We recommend adding this to your .zshrc file
autoload -U compinit && compinit
autoload -U bashcompinit && bashcompinit`

	bashComments := `# To enable bash auto-completion, run: eval "$(` + cmd + ` --bash)"
# We recommend adding this to your .bashrc or .bash_profile file`

	bashScript := `_akamai_cli_bash_autocomplete() {
	local cur opts base
	COMPREPLY=()
	cur="${COMP_WORDS[COMP_CWORD]}"
	opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-shell-completion )
	COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
	return 0
}

complete -F _akamai_cli_bash_autocomplete ` + tools.Self()

	term := terminal.Get(ctx)

	if c.Bool("bash") {
		if _, err = term.Writeln(bashComments); err != nil {
			return err
		}
		if _, err = term.Writeln(bashScript); err != nil {
			return err
		}
		return nil
	}

	if c.Bool("zsh") {
		if _, err = term.Writeln(zshScript); err != nil {
			return err
		}
		if _, err = term.Writeln(bashScript); err != nil {
			return err
		}
		return nil
	}

	cli.ShowAppHelpAndExit(c, 0)
	return nil
}
