package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli"
)

func self() string {
	return filepath.Base(os.Args[0])
}

func getAkamaiCliPath() (string, error) {
	cliHome := os.Getenv("AKAMAI_CLI_HOME")
	if cliHome == "" {
		var err error
		cliHome, err = homedir.Dir()
		if err != nil {
			return "", cli.NewExitError("Package install directory could not be found. Please set $AKAMAI_CLI_HOME.", -1)
		}
	}

	cliPath := filepath.Join(cliHome, ".akamai-cli")
	err := os.MkdirAll(cliPath, 0755)
	if err != nil {
		return "", cli.NewExitError("Unable to create Akamai CLI root directory.", -1)
	}

	return cliPath, nil
}

func getAkamaiCliSrcPath() (string, error) {
	cliHome, _ := getAkamaiCliPath()

	return filepath.Join(cliHome, "src"), nil
}

func getAkamaiCliCachePath() (string, error) {
	if cachePath := getConfigValue("cli", "cache-path"); cachePath != "" {
		return cachePath, nil
	}

	cliHome, _ := getAkamaiCliPath()

	cachePath := filepath.Join(cliHome, "cache")
	err := os.MkdirAll(cachePath, 0775)
	if err != nil {
		return "", err
	}

	setConfigValue("cli", "cache-path", cachePath)
	saveConfig()

	return cachePath, nil
}

func findExec(cmd string) ([]string, error) {
	// "command" becomes: akamai-command, and akamaiCommand
	// "command-name" becomes: akamai-command-name, and akamaiCommandName
	cmdName := "akamai"
	cmdNameTitle := "akamai"
	for _, cmdPart := range strings.Split(cmd, "-") {
		cmdName += "-" + strings.ToLower(cmdPart)
		cmdNameTitle += strings.Title(strings.ToLower(cmdPart))
	}

	systemPath := os.Getenv("PATH")
	packagePaths := getPackageBinPaths()
	os.Setenv("PATH", packagePaths)

	// Quick look for executables on the path
	var path string
	path, err := exec.LookPath(cmdName)
	if err != nil {
		path, err = exec.LookPath(cmdNameTitle)
	}

	if path != "" {
		os.Setenv("PATH", systemPath)
		return []string{path}, nil
	}

	os.Setenv("PATH", systemPath)
	if packagePaths == "" {
		return nil, errors.New("No executables found.")
	}

	for _, path := range filepath.SplitList(packagePaths) {
		filePaths := []string{
			// Search for <path>/akamai-command, <path>/akamaiCommand
			filepath.Join(path, cmdName),
			filepath.Join(path, cmdNameTitle),

			// Search for <path>/akamai-command.*, <path>/akamaiCommand.*
			// This should catch .exe, .bat, .com, .cmd, and .jar
			filepath.Join(path, cmdName+".*"),
			filepath.Join(path, cmdNameTitle+".*"),
		}

		var files []string
		for _, filePath := range filePaths {
			files, _ = filepath.Glob(filePath)
			if len(files) > 0 {
				break
			}
		}

		if len(files) == 0 {
			continue
		}

		cmdFile := files[0]

		packageDir := findPackageDir(filepath.Dir(cmdFile))
		cmdPackage, err := readPackage(packageDir)
		if err != nil {
			return nil, err
		}

		language := determineCommandLanguage(cmdPackage)
		bin := ""
		var cmd []string
		switch {
		// Compiled Languages
		case language == "go" || language == "c#" || language == "csharp":
			err = nil
			cmd = []string{cmdFile}
		case language == "javascript":
			bin, err = exec.LookPath("node")
			if err != nil {
				bin, err = exec.LookPath("nodejs")
			}
			cmd = []string{bin, cmdFile}
		case language == "python":
			var bins pythonBins
			bins, err = findPythonBins(cmdPackage.Requirements.Python)
			bin = bins.python

			cmd = []string{bin, cmdFile}
			// Other languages (php, perl, ruby, etc.)
		default:
			bin, err = exec.LookPath(language)
			cmd = []string{bin, cmdFile}
		}

		if err != nil {
			return nil, err
		}

		return cmd, nil
	}

	return nil, errors.New("No executables found.")
}

func passthruCommand(executable []string) error {
	subCmd := exec.Command(executable[0], executable[1:]...)
	subCmd.Stdin = os.Stdin
	subCmd.Stderr = os.Stderr
	subCmd.Stdout = os.Stdout
	err := subCmd.Run()
	if err != nil {
		return cli.NewExitError("", 1)
	}
	return nil
}

func githubize(repo string) string {
	if strings.HasPrefix(repo, "http") || strings.HasPrefix(repo, "ssh") || strings.HasSuffix(repo, ".git") {
		return strings.TrimPrefix(repo, "ssh://")
	}

	if strings.HasPrefix(repo, "file://") {
		return repo
	}

	if !strings.Contains(repo, "/") {
		repo = "akamai/cli-" + strings.TrimPrefix(repo, "cli-")
	}

	return "https://github.com/" + repo + ".git"
}

func versionCompare(left string, right string) int {
	leftParts := strings.Split(left, ".")
	leftMajor, _ := strconv.Atoi(leftParts[0])
	leftMinor := 0
	leftMicro := 0

	if left == right {
		return 0
	}

	if len(leftParts) > 1 {
		leftMinor, _ = strconv.Atoi(leftParts[1])
	}
	if len(leftParts) > 2 {
		leftMicro, _ = strconv.Atoi(leftParts[2])
	}

	rightParts := strings.Split(right, ".")
	rightMajor, _ := strconv.Atoi(rightParts[0])
	rightMinor := 0
	rightMicro := 0

	if len(rightParts) > 1 {
		rightMinor, _ = strconv.Atoi(rightParts[1])
	}
	if len(rightParts) > 2 {
		rightMicro, _ = strconv.Atoi(rightParts[2])
	}

	if leftMajor > rightMajor {
		return -1
	}

	if leftMajor == rightMajor && leftMinor > rightMinor {
		return -1
	}

	if leftMajor == rightMajor && leftMinor == rightMinor && leftMicro > rightMicro {
		return -1
	}

	return 1
}

func getSpinner(prefix string, finalMsg string) *spinner.Spinner {
	status := spinner.New(spinner.CharSets[26], 500*time.Millisecond)
	status.Prefix = prefix
	status.FinalMSG = finalMsg

	return status
}

func showBanner() {
	fmt.Println()
	bg := color.New(color.BgMagenta)
	bg.Printf(strings.Repeat(" ", 60) + "\n")
	fg := bg.Add(color.FgWhite)
	title := "Welcome to Akamai CLI v" + VERSION
	ws := strings.Repeat(" ", 16)
	fg.Printf(ws + title + ws + "\n")
	bg.Printf(strings.Repeat(" ", 60) + "\n")
	fmt.Println()
}

func setCliTemplates() {
	cli.AppHelpTemplate = "" +
		color.YellowString("Usage: \n") +
		color.BlueString("	{{if .UsageText}}" +
			"{{.UsageText}}" +
			"{{else}}" +
			"{{.HelpName}} " +
			"{{if .VisibleFlags}}[global flags]{{end}}" +
			"{{if .Commands}} command [command flags]{{end}} " +
			"{{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}" +
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
		"{{range .VisibleFlags}}   {{.}}\n\n{{end}}{{end}}" +

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
