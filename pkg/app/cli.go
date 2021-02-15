package app

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/tools"
	"github.com/akamai/cli/pkg/version"

	"github.com/fatih/color"
	"github.com/kardianos/osext"
	"github.com/urfave/cli/v2"
)

const sleepTime24Hours = time.Hour * 24

// CreateApp ...
func CreateApp(ctx context.Context) *cli.App {
	term := terminal.Get(ctx)

	appName := "akamai"
	app := cli.NewApp()
	app.Name = appName
	app.HelpName = appName
	app.Usage = "Akamai CLI"
	app.Version = version.Version
	app.Writer = term
	app.ErrWriter = term.Error()
	app.Copyright = "Copyright (C) Akamai Technologies, Inc"
	app.EnableBashCompletion = true
	app.BashComplete = DefaultAutoComplete
	cli.VersionFlag = &cli.BoolFlag{
		Name:   "version",
		Hidden: true,
	}

	cli.BashCompletionFlag = &cli.BoolFlag{
		Name:   "generate-auto-complete",
		Hidden: true,
	}
	cli.HelpFlag = &cli.BoolFlag{
		Name:  "help",
		Usage: "show help",
	}

	SetHelpTemplates()
	app.Flags = []cli.Flag{
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
			EnvVars: []string{"AKAMAI_CLI_DAEMON"},
		},
	}

	app.Action = func(c *cli.Context) error {
		return defaultAction(c)
	}

	app.Before = func(c *cli.Context) error {
		if c.IsSet("proxy") {
			proxy := c.String("proxy")
			if err := os.Setenv("HTTP_PROXY", proxy); err != nil {
				return err
			}
			if err := os.Setenv("http_proxy", proxy); err != nil {
				return err
			}
			if strings.HasPrefix(proxy, "https") {
				if err := os.Setenv("HTTPS_PROXY", proxy); err != nil {
					return err
				}
				if err := os.Setenv("https_proxy", proxy); err != nil {
					return err
				}
			}
		}

		if c.IsSet("daemon") {
			for {
				time.Sleep(sleepTime24Hours)
			}
		}
		return nil
	}

	return app
}

// DefaultAutoComplete ...
func DefaultAutoComplete(ctx *cli.Context) {
	term := terminal.Get(ctx.Context)
	if ctx.Command.Name == "help" {
		var args []string
		args = append(args, os.Args[0])
		if len(os.Args) > 2 {
			args = append(args, os.Args[2:]...)
		}

		if err := ctx.App.Run(args); err != nil {
			term.WriteError(err.Error())
		}
	}

	commands := make([]*cli.Command, 0)
	flags := make([]cli.Flag, 0)

	if ctx.Command.Name == "" {
		commands = ctx.App.Commands
		flags = ctx.App.Flags
	} else {
		if len(ctx.Command.Subcommands) != 0 {
			commands = ctx.Command.Subcommands
		}

		if len(ctx.Command.Flags) != 0 {
			flags = ctx.Command.Flags
		}
	}

	for _, command := range commands {
		if command.Hidden {
			continue
		}

		for _, name := range command.Names() {
			term.Writeln(ctx.App.Writer, name)
		}
	}

	for _, flag := range flags {
	nextFlag:
		for _, name := range flag.Names() {
			name = strings.TrimSpace(name)

			if len(cli.BashCompletionFlag.Names()) > 0 && name == cli.BashCompletionFlag.Names()[0] {
				continue
			}

			for _, arg := range os.Args {
				if arg == "--"+name || arg == "-"+name {
					continue nextFlag
				}
			}

			switch len(name) {
			case 0:
			case 1:
				term.Writeln(ctx.App.Writer, "-"+name)
			default:
				term.Writeln(ctx.App.Writer, "--"+name)
			}
		}
	}
}

// SetHelpTemplates sets up custom help outputs for app, commands and subcommands
func SetHelpTemplates() {
	cli.AppHelpTemplate = "" +
		color.YellowString("Usage: \n") +
		color.BlueString("	{{if .UsageText}}"+
			"{{.UsageText}}"+
			"{{else}}"+
			"{{.HelpName}} "+
			"{{if .VisibleFlags}}[global flags]{{end}}"+
			"{{if .Commands}} command [command flags]{{end}} "+
			"{{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}"+
			"\n\n{{end}}") +
		"{{if .Description}}\n\n" +
		color.YellowString("Description:\n") +
		"   {{.Description}}" +
		"\n\n{{end}}" +
		"{{if .VisibleCommands}}" +
		color.YellowString("Built-In Commands:\n") +
		"{{range .VisibleCategories}}" +
		"{{if .Name}}" +
		"\n{{.Name}}\n" +
		"{{end}}" +
		"{{range .VisibleCommands}}" +
		color.GreenString("  {{.Name}}") +
		"{{if .Aliases}} ({{ $length := len .Aliases }}{{if eq $length 1}}alias:{{else}}aliases:{{end}} " +
		"{{range $index, $alias := .Aliases}}" +
		"{{if $index}}, {{end}}" +
		color.GreenString("{{$alias}}") +
		"{{end}}" +
		"){{end}}\n" +
		"{{end}}" +
		"{{end}}" +
		"{{end}}\n" +
		"{{if .VisibleFlags}}" +
		color.YellowString("Global Flags:\n") +
		"{{range $index, $option := .VisibleFlags}}" +
		"{{if $index}}\n{{end}}" +
		"   {{$option}}" +
		"{{end}}" +
		"\n\n{{end}}" +
		"{{if .Copyright}}" +
		color.HiBlackString("{{.Copyright}}") +
		"{{end}}\n"

	cli.CommandHelpTemplate = "" +
		color.YellowString("Name: \n") +
		"   {{.HelpName}}\n\n" +
		color.YellowString("Usage: \n") +
		color.BlueString("   {{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}\n\n") +
		"{{if .Category}}" +
		color.YellowString("Type: \n") +
		"   {{.Category}}\n\n{{end}}" +
		"{{if .Description}}" +
		color.YellowString("Description: \n") +
		"   {{.Description}}\n\n{{end}}" +
		"{{if .VisibleFlags}}" +
		color.YellowString("Flags: \n") +
		"{{range .VisibleFlags}}   {{.}}\n{{end}}{{end}}" +
		"{{if .UsageText}}{{.UsageText}}\n{{end}}"

	cli.SubcommandHelpTemplate = "" +
		color.YellowString("Name: \n") +
		"   {{.HelpName}} - {{.Usage}}\n\n" +
		color.YellowString("Usage: \n") +
		color.BlueString("   {{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}\n\n") +
		color.YellowString("Commands:\n") +
		"{{range .VisibleCategories}}" +
		"{{if .Name}}" +
		"{{.Name}}:" +
		"{{end}}" +
		"{{range .VisibleCommands}}" +
		`{{join .Names ", "}}{{"\t"}}{{.Usage}}` +
		"{{end}}\n\n" +
		"{{end}}" +
		"{{if .VisibleFlags}}" +
		color.YellowString("Flags:\n") +
		"{{range .VisibleFlags}}{{.}}\n{{end}}{{end}}"
}

func defaultAction(c *cli.Context) error {
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
    opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-auto-complete )
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

complete -F _akamai_cli_bash_autocomplete ` + tools.Self()

	term := terminal.Get(c.Context)

	if c.Bool("bash") {
		term.Writeln(bashComments)
		term.Writeln(bashScript)
		return nil
	}

	if c.Bool("zsh") {
		term.Writeln(zshScript)
		term.Writeln(bashScript)
		return nil
	}

	cli.ShowAppHelpAndExit(c, 0)
	return nil
}
