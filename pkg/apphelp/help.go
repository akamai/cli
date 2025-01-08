package apphelp

import (
	"embed"
	"errors"
	"io"
	"text/template"

	"github.com/akamai/cli/pkg/autocomplete"
	"github.com/akamai/cli/pkg/tools"

	"github.com/akamai/cli/pkg/color"
	"github.com/urfave/cli/v2"
)

var (
	//go:embed templates/*
	files embed.FS

	// SimplifiedHelpTemplate is the help template with simplified usage and excluded global flags
	SimplifiedHelpTemplate string

	// ErrReadingTemplateFile is returned when reading help template fails
	ErrReadingTemplateFile = errors.New("could not read help template file")
)

func init() {
	tmpl, err := files.ReadFile("templates/simplified_command_help.tmpl")
	if err != nil {
		panic(ErrReadingTemplateFile)
	}
	SimplifiedHelpTemplate = string(tmpl)
}

// Setup sets up custom help outputs and uniforms behavior of help flag and command.
//
// Default help command and flag from urfave/cli have some known discrepancies (https://github.com/urfave/cli/issues/557)
// which are removed by the custom help command that is added by this function. Help flag needs to be added
// manually as it is not appended by the library in case of custom help command being specified.
func Setup(app *cli.App) {
	SetTemplates(app.Flags)
	app.Flags = append(app.Flags, cli.HelpFlag)
	app.Commands = []*cli.Command{
		{
			Name:               "help",
			ArgsUsage:          "[command] [sub-command]",
			Description:        "Displays help information",
			Action:             cmdHelp,
			CustomHelpTemplate: SimplifiedHelpTemplate,
			BashComplete:       autocomplete.Default,
		},
	}
}

// SetTemplates sets up custom help outputs for app, commands and subcommands
func SetTemplates(globalFlags []cli.Flag) {
	tmpl, err := files.ReadFile("templates/app_help.tmpl")
	if err != nil {
		panic(ErrReadingTemplateFile)
	}
	cli.AppHelpTemplate = string(tmpl)

	tmpl, err = files.ReadFile("templates/command_help.tmpl")
	if err != nil {
		panic(ErrReadingTemplateFile)
	}
	cli.CommandHelpTemplate = string(tmpl)

	tmpl, err = files.ReadFile("templates/subcommand_help.tmpl")
	if err != nil {
		panic(ErrReadingTemplateFile)
	}
	cli.SubcommandHelpTemplate = string(tmpl)

	cli.HelpPrinter = makePrintHelp(globalFlags)
}

func makePrintHelp(globalFlags []cli.Flag) func(io.Writer, string, interface{}) {
	type helpData struct {
		GlobalFlags []cli.Flag
		Command     interface{}
	}
	return func(out io.Writer, templ string, data interface{}) {
		funcMap := template.FuncMap{
			"blue":         color.BlueString,
			"green":        color.GreenString,
			"hiBlack":      color.HiBlackString,
			"yellow":       color.YellowString,
			"insertString": tools.InsertAfterNthWord,
		}

		hData := helpData{
			GlobalFlags: globalFlags,
			Command:     data,
		}

		cli.HelpPrinterCustom(out, templ, hData, funcMap)
	}
}
