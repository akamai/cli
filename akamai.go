/*
 * Copyright 2017 Akamai Technologies, Inc
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli"
	"gopkg.in/src-d/go-git.v4"
)

const (
	VERSION = "0.4.2"
)

func main() {
	setCliTemplates()

	os.Setenv("AKAMAI_CLI", "1")

	app := cli.NewApp()
	app.Name = "akamai"
	app.Usage = "Akamai CLI"
	app.Version = VERSION
	app.Copyright = "Copyright (C) Akamai Technologies, Inc"

	firstRun()

	if latestVersion := checkForUpgrade(false); latestVersion != "" {
		if upgradeCli(latestVersion) {
			return
		}
	}

	var builtinCmds map[string]bool = make(map[string]bool)
	for _, cmd := range getBuiltinCommands() {
		builtinCmds[strings.ToLower(cmd.Commands[0].Name)] = true
		app.Commands = append(
			app.Commands,
			cli.Command{
				Name:        strings.ToLower(cmd.Commands[0].Name),
				Aliases:     cmd.Commands[0].Aliases,
				Usage:       cmd.Commands[0].Usage,
				ArgsUsage:   cmd.Commands[0].Arguments,
				Description: cmd.Commands[0].Description,
				Action:      cmd.action,
				UsageText:   cmd.Commands[0].Docs,
				Flags:       cmd.Commands[0].Flags,
			},
		)
	}

	for _, cmd := range getCommands() {
		for _, command := range cmd.Commands {
			if _, ok := builtinCmds[command.Name]; ok {
				continue
			}

			app.Commands = append(
				app.Commands,
				cli.Command{
					Name:        strings.ToLower(command.Name),
					Aliases:     command.Aliases,
					Description: command.Description,

					Action:          cmdSubcommand,
					Category:        color.YellowString("Installed Commands:"),
					SkipFlagParsing: true,
				},
			)
		}
	}

	app.Run(os.Args)
}

func cmdList(c *cli.Context) {
	bold := color.New(color.FgWhite, color.Bold)

	color.Yellow("\nAvailable Commands:\n\n")
	for _, cmd := range getCommands() {
		for _, command := range cmd.Commands {
			bold.Printf("  %s", command.Name)
			if len(command.Aliases) > 0 {
				var aliases string

				if len(command.Aliases) == 1 {
					aliases = "alias"
				} else {
					aliases = "aliases"
				}

				fmt.Printf(" (%s: ", aliases)
				for i, alias := range command.Aliases {
					bold.Print(alias)
					if i < len(command.Aliases)-1 {
						fmt.Print(", ")
					}
				}
				fmt.Print(")")
			}

			fmt.Println()

			fmt.Printf("    %s\n", command.Description)
		}
	}
	fmt.Printf("\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", self()))
}

func cmdInstall(c *cli.Context) error {
	if !c.Args().Present() {
		return cli.NewExitError(color.RedString("You must specify a repository URL"), 1)
	}

	oldCmds := getCommands()

	for _, repo := range c.Args() {
		srcPath, err := getAkamaiCliSrcPath()
		if err != nil {
			return err
		}

		_ = os.MkdirAll(srcPath, 0775)

		repo = githubize(repo)

		fmt.Printf("Attempting to fetch command from %s...", repo)

		dirName := strings.TrimSuffix(filepath.Base(repo), ".git")
		packageDir := srcPath + string(os.PathSeparator) + dirName
		if _, err := os.Stat(packageDir); err == nil {
			fmt.Println("... [" + color.RedString("FAIL") + "]")
			return cli.NewExitError(color.RedString("Package directory already exists (%s)", packageDir), 1)
		}

		_, err = git.PlainClone(packageDir, false, &git.CloneOptions{
			URL:      repo,
			Progress: nil,
		})

		if err != nil {
			fmt.Println("... [" + color.RedString("FAIL") + "]")
			os.RemoveAll(packageDir)
			return cli.NewExitError(color.RedString("Unable to clone repository: "+err.Error()), 1)
		}

		fmt.Println("... [" + color.GreenString("OK") + "]")

		if !installPackage(packageDir, c.Bool("force")) {
			os.RemoveAll(packageDir)
			return cli.NewExitError("", 1)
		}
	}

	listDiff(oldCmds)

	return nil
}

func cmdUpdate(c *cli.Context) error {
	if !c.Args().Present() {
		var builtinCmds map[string]bool = make(map[string]bool)
		for _, cmd := range getBuiltinCommands() {
			builtinCmds[strings.ToLower(cmd.Commands[0].Name)] = true
		}

		for _, cmd := range getCommands() {
			for _, command := range cmd.Commands {
				if _, ok := builtinCmds[command.Name]; !ok {
					if err := updatePackage(command.Name, c.Bool("force")); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	for _, cmd := range c.Args() {
		if err := updatePackage(cmd, c.Bool("force")); err != nil {
			return err
		}
	}

	return nil
}

func cmdUninstall(c *cli.Context) error {
	for _, cmd := range c.Args() {
		exec, err := findExec(cmd)
		if err != nil {
			return cli.NewExitError(color.RedString("Command \"%s\" not found. Try \"%s help\".\n", cmd, self()), 1)
		}

		status := getSpinner(fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd), fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd)+"... ["+color.GreenString("OK")+"]\n")
		status.Start()

		var repoDir string
		if len(exec) == 1 {
			repoDir = findPackageDir(filepath.Dir(exec[0]))
		} else if len(exec) > 1 {
			repoDir = findPackageDir(filepath.Dir(exec[len(exec)-1]))
		}

		if repoDir == "" {
			status.FinalMSG = fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd) + "... [" + color.RedString("FAIL") + "]\n"
			status.Stop()
			return cli.NewExitError(color.RedString("unable to uninstall, was it installed using "+color.CyanString("\"akamai install\"")+"?"), 1)
		}

		if err := os.RemoveAll(repoDir); err != nil {
			status.FinalMSG = fmt.Sprintf("Attempting to uninstall \"%s\" command...", cmd) + "... [" + color.RedString("FAIL") + "]\n"
			status.Stop()
			return cli.NewExitError(color.RedString("unable to remove directory: %s", repoDir), 1)
		}

		status.Stop()
	}

	return nil
}

func cmdUpgrade(c *cli.Context) error {
	status := getSpinner("Checking for upgrades...", "Checking for upgrades...... ["+color.GreenString("OK")+"]\n")

	status.Start()
	if latestVersion := checkForUpgrade(true); latestVersion != "" {
		status.Stop()
		fmt.Printf("Found new version: %s (current version: %s)\n", color.BlueString("v"+latestVersion), color.BlueString("v"+VERSION))
		os.Args = []string{os.Args[0], "--version"}
		upgradeCli(latestVersion)
	} else {
		status.FinalMSG = "Checking for upgrades...... [" + color.CyanString("OK") + "]\n"
		status.Stop()
		fmt.Printf("Akamai CLI (%s) is already up-to-date", color.CyanString("v"+VERSION))
	}

	return nil
}

func cmdSubcommand(c *cli.Context) error {
	cachePath, err := getAkamaiCliCachePath()
	if err != nil {
		return cli.NewExitError("Unable to determine cache path.", 1)
	}
	os.Setenv("AKAMAI_CLI_CACHE_DIR", cachePath)

	cmd := c.Command.Name

	executable, err := findExec(cmd)
	if err != nil {
		return cli.NewExitError(color.RedString("Executable \"%s\" not found.", cmd), 1)
	}

	var packageDir string
	if len(executable) == 1 {
		packageDir = findPackageDir(filepath.Dir(executable[0]))
	} else if len(executable) > 1 {
		packageDir = findPackageDir(filepath.Dir(executable[1]))
	}

	cmdPackage, _ := readPackage(packageDir)

	if cmdPackage.Requirements.Python != "" {
		err = setPythonPath(packageDir)
		if err != nil {
			return err
		}
	}

	executable = append(executable, os.Args[2:]...)
	return passthruCommand(executable)
}

func cmdHelp(c *cli.Context) error {
	if c.Args().Present() {
		cmd := c.Args().First()

		builtinCmds := getBuiltinCommands()
		for _, builtInCmd := range builtinCmds {
			if builtInCmd.Commands[0].Name == cmd {
				return cli.ShowCommandHelp(c, cmd)
			}
		}

		// The arg mangling ensures that aliases are handled
		os.Args = append([]string{os.Args[0], cmd, "help"}, c.Args().Tail()...)
		main()
		return nil
	}

	return cli.ShowAppHelp(c)
}

func getBuiltinCommands() []commandPackage {
	commands := []commandPackage{
		{
			Commands: []Command{
				{
					Name:        "help",
					Arguments:   "[command] [sub-command]",
					Description: "Displays help information",
				},
			},
			action: cmdHelp,
		},
		{
			Commands: []Command{
				{
					Name:        "list",
					Description: "Displays available commands",
				},
			},
			action: cmdList,
		},
		{
			Commands: []Command{
				{
					Name:      "install",
					Arguments: "<package name or repository URL>...",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "force",
							Usage: "Force binary installation if available when source installation fails",
						},
					},
					Description: "Fetch and install packages from a Git repository.",
					Docs:        "Examples:\n\n   akamai install property purge\n   akamai install akamai/cli-property\n   akamai install git@github.com:akamai/cli-property.git\n   akamai install https://github.com/akamai/cli-property.git",
				},
			},
			action: cmdInstall,
		},
		{
			Commands: []Command{
				{
					Name:        "uninstall",
					Arguments:   "<command>...",
					Description: "Uninstall package containing <command>",
				},
			},
			action: cmdUninstall,
		},
		{
			Commands: []Command{
				{
					Name:      "update",
					Arguments: "[<command>...]",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "force",
							Usage: "Force binary installation if available when source installation fails",
						},
					},
					Description: "Update one or more commands. If no command is specified, all commands are updated",
				},
			},
			action: cmdUpdate,
		},
	}

	upgradeCommand := getUpgradeCommand()
	if upgradeCommand != nil {
		commands = append(commands, *upgradeCommand)
	}

	return commands
}

func getCommands() []commandPackage {
	var commands []commandPackage
	var commandMap map[string]bool = make(map[string]bool)

	for _, cmd := range getBuiltinCommands() {
		commandMap[cmd.Commands[0].Name] = true
		commands = append(commands, cmd)
	}

	packagePaths := getPackagePaths()
	if packagePaths == "" {
		return commands
	}

	for _, dir := range filepath.SplitList(packagePaths) {
		cmdPackage, err := readPackage(dir)
		if err == nil {
			commands = append(commands, cmdPackage)
		}
	}

	return commands
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

	cliPath := cliHome + string(os.PathSeparator) + ".akamai-cli"
	err := os.MkdirAll(cliPath, 0755)
	if err != nil {
		return "", cli.NewExitError("Unable to create Akamai CLI root directory.", -1)
	}

	return cliPath, nil
}

func getAkamaiCliSrcPath() (string, error) {
	cliHome, _ := getAkamaiCliPath()

	return cliHome + string(os.PathSeparator) + "src", nil
}

func getAkamaiCliCachePath() (string, error) {
	cliHome, _ := getAkamaiCliPath()

	cachePath := cliHome + string(os.PathSeparator) + ".akamai-cli" + string(os.PathSeparator) + "cache"
	err := os.MkdirAll(cachePath, 0775)
	if err != nil {
		return "", err
	}

	return cachePath, nil
}

func getPackagePaths() string {
	path := ""
	akamaiCliPath, err := getAkamaiCliSrcPath()
	if err == nil && akamaiCliPath != "" {
		paths, _ := filepath.Glob(akamaiCliPath + string(os.PathSeparator) + "*")
		if len(paths) > 0 {
			path += strings.Join(paths, string(os.PathListSeparator))
		}
	}

	return path
}

func getPackageBinPaths() string {
	path := ""
	akamaiCliPath, err := getAkamaiCliSrcPath()
	if err == nil && akamaiCliPath != "" {
		paths, _ := filepath.Glob(akamaiCliPath + string(os.PathSeparator) + "*")
		if len(paths) > 0 {
			path += strings.Join(paths, string(os.PathListSeparator))
		}
		paths, _ = filepath.Glob(akamaiCliPath + string(os.PathSeparator) + "*" + string(os.PathSeparator) + "bin")
		if len(paths) > 0 {
			path += string(os.PathListSeparator) + strings.Join(paths, string(os.PathListSeparator))
		}
	}

	return path
}

func listDiff(oldcmds []commandPackage) {
	cmds := getCommands()

	var old []Command
	for _, oldcmd := range oldcmds {
		for _, cmd := range oldcmd.Commands {
			old = append(old, cmd)
		}
	}

	var new []Command
	for _, newcmd := range cmds {
		for _, cmd := range newcmd.Commands {
			new = append(new, cmd)
		}
	}

	var unchanged map[string]bool = make(map[string]bool)
	var added map[string]bool = make(map[string]bool)
	var removed map[string]bool = make(map[string]bool)

	for _, newCmd := range new {
		found := false
		for _, oldCmd := range old {
			if newCmd.Name == oldCmd.Name {
				found = true
				unchanged[newCmd.Name] = true
				break
			}
		}

		if !found {
			added[newCmd.Name] = true
		}
	}

	for _, oldCmd := range old {
		found := false
		for _, newCmd := range new {
			if newCmd.Name == oldCmd.Name {
				found = true
				break
			}
		}

		if !found {
			removed[oldCmd.Name] = true
		}
	}

	bold := color.New(color.FgWhite, color.Bold)

	color.Yellow("\nAvailable Commands:\n\n")
	for _, cmd := range getCommands() {
		for _, command := range cmd.Commands {
			if _, ok := unchanged[command.Name]; ok {
				bold.Printf("  %s", command.Name)
			} else if _, ok := added[command.Name]; ok {
				fmt.Print(color.GreenString("  %s", command.Name))
			} else if _, ok := removed[command.Name]; ok {
				fmt.Print(color.RedString("  %s", command.Name))
			}
			if len(command.Aliases) > 0 {
				var aliases string

				if len(command.Aliases) == 1 {
					aliases = "alias"
				} else {
					aliases = "aliases"
				}

				fmt.Printf(" (%s: ", aliases)
				for i, alias := range command.Aliases {
					if _, ok := unchanged[command.Name]; ok {
						bold.Print(alias)
					} else if _, ok := added[command.Name]; ok {
						fmt.Print(color.GreenString(alias))
					} else if _, ok := removed[command.Name]; ok {
						fmt.Print(color.RedString(alias))
					}

					if i < len(command.Aliases)-1 {
						fmt.Print(", ")
					}
				}
				fmt.Print(")")
			}

			fmt.Println()

			fmt.Printf("    %s\n", command.Description)
		}
	}
	fmt.Printf("\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", self()))
}

type commandPackage struct {
	Commands []Command `json:"commands"`

	Requirements struct {
		Go     string `json:"go"`
		Php    string `json:"php"`
		Node   string `json:"node"`
		Ruby   string `json:"ruby"`
		Python string `json:"python"`
	} `json:"requirements"`

	action interface{}
}

type Command struct {
	Name        string     `json:"name"`
	Aliases     []string   `json:"aliases"`
	Version     string     `json:"version"`
	Description string     `json:"description"`
	Usage       string     `json:"usage"`
	Docs        string     `json:-`
	Arguments   string     `json:"arguments"`
	Flags       []cli.Flag `json:"-"`
	Bin         string     `json:"bin"`
	BinSuffix   string     `json:"-"`
	OS          string     `json:"-"`
	Arch        string     `json:"-"`
}

func readPackage(dir string) (commandPackage, error) {
	if _, err := os.Stat(dir + string(os.PathSeparator) + "/cli.json"); err != nil {
		dir = path.Dir(dir)
		if _, err = os.Stat(dir + string(os.PathSeparator) + "/cli.json"); err != nil {
			return commandPackage{}, cli.NewExitError("Package does not contain a cli.json file.", 1)
		}
	}

	var packageData commandPackage
	cliJson, err := ioutil.ReadFile(dir + string(os.PathSeparator) + "/cli.json")
	if err != nil {
		return commandPackage{}, err
	}

	err = json.Unmarshal(cliJson, &packageData)
	if err != nil {
		return commandPackage{}, err
	}

	for key := range packageData.Commands {
		packageData.Commands[key].Name = strings.ToLower(packageData.Commands[key].Name)
	}

	return packageData, nil
}

func installPackage(dir string, forceBinary bool) bool {
	status := getSpinner("Installing...", "Installing...... ["+color.GreenString("OK")+"]\n")

	status.Start()
	cmdPackage, err := readPackage(dir)

	if err != nil {
		status.FinalMSG = "Installing...... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		fmt.Println(err.Error())
		return false
	}

	lang := determineCommandLanguage(cmdPackage)
	if lang == "" {
		status.FinalMSG = "Installing...... [" + color.BlueString("OK") + "]\n"
		status.Stop()
		color.Blue("Package installed successfully, however package type is unknown, and may or may not function correctly.")
		return true
	}

	var success bool
	switch lang {
	case "php":
		success, err = installPHP(dir, cmdPackage)
	case "javascript":
		success, err = installJavaScript(dir, cmdPackage)
	case "ruby":
		success, err = installRuby(dir, cmdPackage)
	case "python":
		success, err = installPython(dir, cmdPackage)
	case "go":
		success, err = installGolang(dir, cmdPackage)
	default:
		success = false
		err = nil
	}

	if success == false || err != nil {
		first := true
		for _, cmd := range cmdPackage.Commands {
			if cmd.Bin != "" {
				if first {
					first = false
					status.FinalMSG = "Installing...... [" + color.CyanString("WARN") + "]\n"
					status.Stop()
					color.Cyan(err.Error())
					if !forceBinary {
						if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
							return false
						}

						fmt.Print("Binary command(s) found, would you like to try download and install it? (Y/n): ")
						answer := ""
						fmt.Scanln(&answer)
						if answer != "" && strings.ToLower(answer) != "y" {
							return false
						}
					}

					os.MkdirAll(dir+string(os.PathSeparator)+"bin", 0775)
				}

				status := getSpinner("Downloading binary...", "Downloading binary...... ["+color.GreenString("OK")+"]\n")
				status.Start()
				if !downloadBin(dir+string(os.PathSeparator)+"bin", cmd) {
					status.FinalMSG = "Downloading binary...... [" + color.RedString("FAIL") + "]\n"
					status.Stop()
					color.Red("Unable to download binary")
					return false
				}
				success = true
				err = nil
			}
		}
	}

	if err != nil {
		status.FinalMSG = "Downloading binary...... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		color.Red(err.Error())
		return false
	}

	if success {
		status.Stop()
		return true
	}

	status.FinalMSG = "Downloading binary...... [" + color.CyanString("OK") + "]\n"
	status.Stop()
	return true
}

func updatePackage(cmd string, forceBinary bool) error {
	exec, err := findExec(cmd)
	if err != nil {
		return cli.NewExitError(color.RedString("Command \"%s\" not found. Try \"%s help\".\n", cmd, self()), 1)
	}

	status := getSpinner(fmt.Sprintf("Attempting to update \"%s\" command...", cmd), fmt.Sprintf("Attempting to update \"%s\" command...", cmd)+"... ["+color.CyanString("OK")+"]\n")
	status.Start()

	var repoDir string
	if len(exec) == 1 {
		repoDir = findPackageDir(filepath.Dir(exec[0]))
	} else if len(exec) > 1 {
		repoDir = findPackageDir(filepath.Dir(exec[len(exec)-1]))
	}

	if repoDir == "" {
		status.FinalMSG = fmt.Sprintf("Attempting to update \"%s\" command...", cmd) + "... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		return cli.NewExitError(color.RedString("unable to update, was it installed using "+color.CyanString("\"akamai install\"")+"?"), 1)
	}

	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return err
	}

	err = repo.Fetch(&git.FetchOptions{
		RemoteName: git.DefaultRemoteName,
	})

	if err != nil && err.Error() != "already up-to-date" {
		status.FinalMSG = fmt.Sprintf("Attempting to update \"%s\" command...", cmd) + "... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		return cli.NewExitError("Unable to fetch updates", 1)
	}

	workdir, _ := repo.Worktree()
	ref, err := repo.Reference("refs/remotes/"+git.DefaultRemoteName+"/master", true)
	if err != nil {
		status.FinalMSG = fmt.Sprintf("Attempting to update \"%s\" command...", cmd) + "... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		return cli.NewExitError("Unable to update command", 1)
	}

	head, _ := repo.Head()
	if head.Hash() == ref.Hash() {
		status.FinalMSG = fmt.Sprintf("Attempting to update \"%s\" command...", cmd) + "... [" + color.CyanString("OK") + "]\n"
		status.Stop()
		color.Cyan("command \"%s\" already up-to-date", cmd)
		return nil
	}

	err = workdir.Checkout(&git.CheckoutOptions{
		Branch: ref.Name(),
		Force:  true,
	})

	if err != nil {
		status.FinalMSG = fmt.Sprintf("Attempting to update \"%s\" command...", cmd) + "... [" + color.RedString("FAIL") + "]\n"
		status.Stop()
		return cli.NewExitError("Unable to update command", 1)
	}

	status.Stop()

	if !installPackage(repoDir, forceBinary) {
		return cli.NewExitError("Unable to update command", 1)
	}

	return nil
}

func installPHP(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("php")
	if err != nil {
		return false, cli.NewExitError("Unable to locate PHP runtime", 1)
	}

	if cmdPackage.Requirements.Php != "" && cmdPackage.Requirements.Php != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		r, _ := regexp.Compile("PHP (.*?) .*")
		matches := r.FindStringSubmatch(string(output))
		if len(matches) == 0 {
			return false, cli.NewExitError(fmt.Sprintf("PHP %s is required to install this command. Unable to determine installed version.", cmdPackage.Requirements.Php), 1)
		}

		if versionCompare(cmdPackage.Requirements.Php, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("PHP %s is required to install this command.", cmdPackage.Requirements.Php), 1)
		}
	}

	if _, err := os.Stat(dir + string(os.PathSeparator) + "/composer.json"); err == nil {
		if _, err := os.Stat(dir + string(os.PathSeparator) + "/composer.phar"); err == nil {
			cmd := exec.Command(bin, dir+string(os.PathSeparator)+"/composer.phar", "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}

		bin, err := exec.LookPath("composer")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}

		bin, err = exec.LookPath("composer.phar")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}

		return false, cli.NewExitError("Unable to find package manager.", 1)
	}

	return false, nil
}

func installJavaScript(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("node")
	if err != nil {
		bin, err = exec.LookPath("nodejs")
		if err != nil {
			return false, cli.NewExitError(("Unable to locate Node.js runtime"), 1)
		}
	}

	if cmdPackage.Requirements.Node != "" && cmdPackage.Requirements.Node != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		r, _ := regexp.Compile("^v(.*?)\\s*$")
		matches := r.FindStringSubmatch(string(output))
		if versionCompare(cmdPackage.Requirements.Node, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("Node.js %s is required to install this command.", cmdPackage.Requirements.Node), 1)
		}
	}

	if _, err := os.Stat(dir + string(os.PathSeparator) + "/yarn.lock"); err == nil {
		bin, err := exec.LookPath("yarn")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	if _, err := os.Stat(dir + string(os.PathSeparator) + "/package.json"); err == nil {
		bin, err := exec.LookPath("npm")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, cli.NewExitError("Unable to find package manager.", 1)
}

func installRuby(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("ruby")
	if err != nil {
		return false, cli.NewExitError(("Unable to locate Ruby runtime"), 1)
	}

	if cmdPackage.Requirements.Ruby != "" && cmdPackage.Requirements.Ruby != "*" {
		cmd := exec.Command(bin, "-v")
		output, _ := cmd.Output()
		r, _ := regexp.Compile("^ruby (.*?)(p.*?) (.*)")
		matches := r.FindStringSubmatch(string(output))
		if versionCompare(cmdPackage.Requirements.Ruby, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("Ruby %s is required to install this command.", cmdPackage.Requirements.Ruby), 1)
		}
	}

	if _, err := os.Stat(dir + string(os.PathSeparator) + "/Gemfile"); err == nil {
		bin, err := exec.LookPath("bundle")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, cli.NewExitError("Unable to find package manager.", 1)
}

func installPython(dir string, cmdPackage commandPackage) (bool, error) {
	var (
		bin string
		err error
	)

	if cmdPackage.Requirements.Python != "" && cmdPackage.Requirements.Python != "*" {
		if versionCompare("3.0.0", cmdPackage.Requirements.Python) != -1 {
			bin, err = exec.LookPath("python3")
			if err != nil {
				bin, err = exec.LookPath("python")
				if err != nil {
					return false, cli.NewExitError("Unable to locate Python 3 runtime", 1)
				}
			}
		} else {
			bin, err = exec.LookPath("python2")
			if err != nil {
				bin, err = exec.LookPath("python")
				if err != nil {
					bin, err = exec.LookPath("python3")
					if err != nil {
						return false, cli.NewExitError("Unable to locate Python runtime", 1)
					}
				}
			}
		}
	} else {
		bin, err = exec.LookPath("python3")
		if err != nil {
			bin, err = exec.LookPath("python2")
			if err != nil {
				bin, err = exec.LookPath("python")
				if err != nil {
					return false, cli.NewExitError("Unable to locate Python runtime", 1)
				}
			}
		}
	}

	if cmdPackage.Requirements.Python != "" && cmdPackage.Requirements.Python != "*" {
		cmd := exec.Command(bin, "--version")
		output, _ := cmd.CombinedOutput()
		r, _ := regexp.Compile(`Python (\d+\.\d+\.\d+).*`)
		matches := r.FindStringSubmatch(string(output))
		if versionCompare(cmdPackage.Requirements.Python, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("Python %s is required to install this command.", cmdPackage.Requirements.Python), 1)
		}
	}

	if _, err := os.Stat(dir + string(os.PathSeparator) + "/requirements.txt"); err == nil {
		if cmdPackage.Requirements.Python != "" && cmdPackage.Requirements.Python != "*" {
			if versionCompare("3.0.0", cmdPackage.Requirements.Python) != -1 {
				bin, err = exec.LookPath("pip3")
				if err != nil {
					bin, err = exec.LookPath("pip")
					if err != nil {
						return false, cli.NewExitError("Unable to find package manager.", 1)
					}
				}
			} else {
				bin, err = exec.LookPath("pip2")
				if err != nil {
					bin, err = exec.LookPath("pip")
					if err != nil {
						bin, err = exec.LookPath("pip3")
						if err != nil {
							return false, cli.NewExitError("Unable to find package manager.", 1)
						}
					}
				}
			}
		} else {
			bin, err = exec.LookPath("pip3")
			if err != nil {
				bin, err = exec.LookPath("pip2")
				if err != nil {
					bin, err = exec.LookPath("pip")
					if err != nil {
						return false, cli.NewExitError("Unable to find package manager.", 1)
					}
				}
			}
		}

		if err == nil {
			if runtime.GOOS != "windows" {
				systemHome := os.Getenv("HOME")
				os.Setenv("HOME", dir)
				cmd := exec.Command(bin, "install", "--user", "-r", "requirements.txt")
				cmd.Dir = dir
				err = cmd.Run()
				os.Setenv("HOME", systemHome)
			} else {
				cmd := exec.Command(bin, "install", "--isolated", "--prefix", dir, "-r", "requirements.txt")
				cmd.Dir = dir
				err = cmd.Run()
			}
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return true, nil
}

func installGolang(dir string, cmdPackage commandPackage) (bool, error) {
	bin, err := exec.LookPath("go")
	if err != nil {
		return false, cli.NewExitError("Unable to locate Go runtime", 1)
	}

	if cmdPackage.Requirements.Go != "" && cmdPackage.Requirements.Go != "*" {
		cmd := exec.Command(bin, "version")
		output, _ := cmd.Output()
		r, _ := regexp.Compile("go version go(.*?) .*")
		matches := r.FindStringSubmatch(string(output))
		if versionCompare(cmdPackage.Requirements.Go, matches[1]) == -1 {
			return false, cli.NewExitError(fmt.Sprintf("Go %s is required to install this command.", cmdPackage.Requirements.Go), 1)
		}
	}

	goPath, err := homedir.Dir()
	if err != nil {
		return false, cli.NewExitError(color.RedString("Unable to determine home directory"), 1)
	}
	goPath += string(os.PathSeparator) + ".akamai-cli"
	os.Setenv("GOPATH", os.Getenv("GOPATH")+string(os.PathListSeparator)+goPath)

	if _, err := os.Stat(dir + string(os.PathSeparator) + "glide.lock"); err == nil {
		bin, err := exec.LookPath("glide")
		if err == nil {
			cmd := exec.Command(bin, "install")
			cmd.Dir = dir
			err = cmd.Run()
			if err != nil {
				return false, cli.NewExitError(err.Error(), 1)
			}
		} else {
			return false, cli.NewExitError("Unable to find package manager.", 1)
		}
	}

	execName := "akamai-" + strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(path.Base(dir), "akamai-"), "cli-"))

	cmd := exec.Command(bin, "build", "-o", execName, ".")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		return false, cli.NewExitError(err.Error(), 1)
	}

	return true, nil
}

func determineCommandLanguage(cmdPackage commandPackage) string {
	if cmdPackage.Requirements.Php != "" {
		return "php"
	}

	if cmdPackage.Requirements.Node != "" {
		return "javascript"
	}

	if cmdPackage.Requirements.Ruby != "" {
		return "ruby"
	}

	if cmdPackage.Requirements.Go != "" {
		return "go"
	}

	if cmdPackage.Requirements.Python != "" {
		return "python"
	}

	return ""
}

func downloadBin(dir string, cmd Command) bool {
	cmd.Arch = runtime.GOARCH

	cmd.OS = runtime.GOOS
	if runtime.GOOS == "darwin" {
		cmd.OS = "mac"
	}

	if runtime.GOOS == "windows" {
		cmd.BinSuffix = ".exe"
	}

	t := template.Must(template.New("url").Parse(cmd.Bin))
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, cmd); err != nil {
		return false
	}

	url := buf.String()

	bin, err := os.Create(dir + string(os.PathSeparator) + "akamai-" + strings.ToLower(cmd.Name) + cmd.BinSuffix)
	bin.Chmod(0775)
	if err != nil {
		return false
	}
	defer bin.Close()

	res, err := http.Get(url)
	if err != nil {
		return false
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return false
	}

	n, err := io.Copy(bin, res.Body)
	if err != nil || n == 0 {
		return false
	}

	return true
}

func setPythonPath(packageDir string) error {
	var pythonPath string

	if runtime.GOOS == "linux" {
		packageDir += string(os.PathSeparator) + ".local" + string(os.PathSeparator) + "lib" + string(os.PathSeparator) + "python*"
	} else if runtime.GOOS == "darwin" {
		packageDir += string(os.PathSeparator) + "Library" + string(os.PathSeparator) + "Python" + string(os.PathSeparator) + "*"
	} else if runtime.GOOS == "windows" {
		packageDir += string(os.PathSeparator) + "Lib"
	}

	pythonPaths, err := filepath.Glob(packageDir)
	if err != nil {
		return err
	}

	if len(pythonPaths) > 0 {
		pythonPath = pythonPaths[0] + string(os.PathSeparator) + "site-packages"
	}

	systemPythonPath := os.Getenv("PYTHONPATH")
	if systemPythonPath == "" {
		bin, _ := exec.LookPath("python")
		cmd := exec.Command(bin, "-c", "import sys, os; print(os.pathsep.join(sys.path))")
		output, _ := cmd.CombinedOutput()
		systemPythonPath = strings.Trim(string(output), "\r\n")
	}

	pythonPath = string(os.PathListSeparator) + pythonPath
	if systemPythonPath != "" {
		pythonPath += string(os.PathListSeparator) + strings.TrimPrefix(systemPythonPath, string(os.PathListSeparator))
	}

	if len(pythonPath) == 0 {
		return cli.NewExitError(color.RedString("Unable to determine package path."), 1)
	}

	os.Setenv("PYTHONPATH", pythonPath)

	return nil
}

func self() string {
	return path.Base(os.Args[0])
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
			path + string(os.PathSeparator) + cmdName,
			path + string(os.PathSeparator) + cmdNameTitle,

			// Search for <path>/akamai-command.*, <path>/akamaiCommand.*
			// This should catch .exe, .bat, .com, .cmd, and .jar
			path + string(os.PathSeparator) + cmdName + ".*",
			path + string(os.PathSeparator) + cmdNameTitle + ".*",
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
		cmd := []string{}
		switch {
		// Compiled Languages
		case language == "go" || language == "c#" || language == "csharp":
			err = nil
			cmd = []string{cmdFile}
			// Node is special
		case language == "javascript":
			bin, err = exec.LookPath("node")
			if err != nil {
				bin, err = exec.LookPath("nodejs")
			}
			cmd = []string{bin, cmdFile}
		case language == "python":
			if versionCompare("3.0.0", cmdPackage.Requirements.Python) != -1 {
				bin, err = exec.LookPath("python3")
				if err != nil {
					bin, err = exec.LookPath("python")
				}
			} else {
				bin, err = exec.LookPath("python2")
				if err != nil {
					bin, err = exec.LookPath("python")
				}
			}
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

	if !strings.Contains(repo, "/") {
		repo = "akamai/cli-" + strings.TrimPrefix(repo, "cli-")
	}

	return "https://github.com/" + repo + ".git"
}

func findPackageDir(dir string) string {
	if _, err := os.Stat(dir + string(os.PathSeparator) + ".git"); err != nil {
		if os.IsNotExist(err) {
			if path.Dir(dir) == "" {
				return ""
			}

			return findPackageDir(filepath.Dir(dir))
		}
	}

	return dir
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
