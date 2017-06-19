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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"errors"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli"
	"gopkg.in/src-d/go-git.v4"
	"runtime"
)

func main() {
	setCliTemplates()

	os.Setenv("AKAMAI_CLI", "1")

	app := cli.NewApp()
	app.Name = "akamai"
	app.Usage = "Akamai CLI"
	app.Version = "0.1.0"
	app.Copyright = "Copyright (C) Akamai Technologies, Inc"
	app.Authors = []cli.Author{{
		Name:  "Davey Shafik",
		Email: "dshafik@akamai.com",
	}}

	helpInfo := getHelp()

	for _, cmd := range helpInfo {
		app.Commands = append(
			app.Commands,
			cli.Command{
				Name:      strings.ToLower(cmd.Commands[0].Name),
				Usage:     cmd.Commands[0].Usage,
				ArgsUsage: cmd.Commands[0].Arguments,
				Action:    cmd.action,
			},
		)
	}

	for _, cmd := range getCommands() {
		for _, command := range cmd.Commands {
			if _, ok := helpInfo[command.Name]; ok {
				continue
			}

			app.Commands = append(
				app.Commands,
				cli.Command{
					Name:            strings.ToLower(command.Name),
					Usage:           command.Usage,
					ArgsUsage:       command.Arguments,
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
	color.Yellow("\nAvailable Commands:")
	for _, cmd := range getCommands() {
		for _, command := range cmd.Commands {
			fmt.Println("  " + command.Name + "\t" + command.Description)
		}
	}
	fmt.Printf("\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", self()))
}

func cmdGet(c *cli.Context) error {
	if !c.Args().Present() {
		return cli.NewExitError(color.RedString("You must specify a repository URL"), 1)
	}

	repo := c.Args().First()

	srcPath, err := homedir.Dir()
	if err != nil {
		return cli.NewExitError(color.RedString("Unable to determine home directory"), 1)
	}
	srcPath += string(os.PathSeparator) + ".akamai-cli" + string(os.PathSeparator) + "src"
	_ = os.MkdirAll(srcPath, 0775)

	oldCmds := getCommands()

	repo = githubize(repo)

	fmt.Print("Attempting to fetch command...")

	dirName := strings.TrimSuffix(filepath.Base(repo), ".git")

	_, err = git.PlainClone(srcPath+string(os.PathSeparator)+dirName, false, &git.CloneOptions{
		URL:      repo,
		Progress: nil,
	})

	if err != nil {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		os.RemoveAll(srcPath + string(os.PathSeparator) + dirName)
		return cli.NewExitError(color.RedString("Unable to clone repository: "+err.Error()), 1)
	}

	fmt.Println("... [" + color.GreenString("OK") + "]")

	if !installPackage(srcPath + string(os.PathSeparator) + dirName) {
		os.RemoveAll(srcPath + string(os.PathSeparator) + dirName)
		return cli.NewExitError("", 1)
	}

	listDiff(oldCmds)

	return nil
}

func cmdUpdate(c *cli.Context) error {
	cmd := c.Args().First()

	if cmd == "" {
		help := getHelp()
		for _, cmd := range getCommands() {
			for _, command := range cmd.Commands {
				if _, ok := help[command.Name]; !ok {
					if err := update(command.Name); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	return update(cmd)
}

func cmdSubcommand(c *cli.Context) error {
	cmd := c.Command.Name

	executable, err := findExec(cmd)
	if err != nil {
		return cli.NewExitError(color.RedString("Executable \"%s\" not found.", cmd), 1)
	}

	var packageDir string
	if len(executable) == 1 {
		packageDir = findPackageDir(executable[0])
	} else if len(executable) > 1 {
		packageDir = findPackageDir(executable[1])
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

		help := getHelp()
		if _, ok := help[cmd]; !ok {
			args := append([]string{"help"}, c.Args().Tail()...)

			executable, err := findExec(cmd)
			if err != nil {
				return err
			}

			executable = append(executable, args...)
			return passthruCommand(executable)
		}

		return cli.ShowCommandHelp(c, cmd)
	}

	return cli.ShowAppHelp(c)
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

func self() string {
	return path.Base(os.Args[0])
}

type help map[string]commandPackage

func getHelp() help {
	return help{
		"help": {
			Commands: []Command{
				{
					Name:        "help",
					Usage:       "[command] [sub-command]",
					Description: "Displays help information",
				},
			},
			action: cmdHelp,
		},
		"list": {
			Commands: []Command{
				{
					Name:        "list",
					Description: "Displays available commands",
				},
			},
			action: cmdList,
		},
		"get": {
			Commands: []Command{
				{
					Name:        "get",
					Usage:       "<repository URL>",
					Description: "Fetch and install a sub-command from a Git repository",
				},
			},
			action: cmdGet,
		},
		"update": {
			Commands: []Command{
				{
					Name:        "update",
					Usage:       "[command]",
					Description: "Update a sub-command. If no command is specified, all commands are updated",
				},
			},
			action: cmdUpdate,
		},
	}
}

func getCommands() []commandPackage {
	var commands []commandPackage
	var commandMap map[string]bool = make(map[string]bool)

	for _, cmd := range getHelp() {
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

func getPackageCommands(dir string) commandPackage {
	var command commandPackage

	cmdPath := filepath.Join(dir, "akamai*")
	matches, err := filepath.Glob(cmdPath)
	if err != nil {
		return commandPackage{}
	}

	for _, match := range matches {
		_, err := exec.LookPath(match)
		if err == nil {
			name := strings.ToLower(
				strings.TrimSuffix(
					strings.TrimPrefix(
						strings.TrimPrefix(
							path.Base(match),
							"akamai-",
						),
						"akamai"),
					".exe",
				),
			)
			if len(name) != 0 {
				command.Commands = append(command.Commands, Command{
					Name: name,
				})
			}
		}
	}

	return command
}

func getAkamaiCliPath() string {
	homedir, err := homedir.Dir()
	if err == nil {
		return homedir + string(os.PathSeparator) + ".akamai-cli" + string(os.PathSeparator) + "src"
	}

	return ""
}

func getPackagePaths() string {
	path := ""
	akamaiCliPath := getAkamaiCliPath()
	if akamaiCliPath != "" {
		paths, _ := filepath.Glob(akamaiCliPath + string(os.PathSeparator) + "*")
		if len(paths) > 0 {
			path += strings.Join(paths, string(os.PathListSeparator))
		}
	}

	return path
}

func getPackageBinPaths() string {
	path := ""
	akamaiCliPath := getAkamaiCliPath()
	if akamaiCliPath != "" {
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
		// Other languages (php, perl, ruby, python, java, etc.)
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

func update(cmd string) error {
	exec, err := findExec(cmd)
	if err != nil {
		return cli.NewExitError(color.RedString("Command \"%s\" not found. Try \"%s help\".\n", cmd, self()), 1)
	}

	fmt.Printf("Attempting to update \"%s\" command...", cmd)

	var repoDir string
	if len(exec) == 1 {
		repoDir = findPackageDir(filepath.Dir(exec[0]))
	} else if len(exec) > 1 {
		repoDir = findPackageDir(filepath.Dir(exec[len(exec)-1]))
	}

	if repoDir == "" {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		return cli.NewExitError(color.RedString("unable to update, was it installed using "+color.CyanString("\"akamai get\"")+"?"), 1)
	}

	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return err
	}

	err = repo.Fetch(&git.FetchOptions{
		RemoteName: git.DefaultRemoteName,
	})

	if err != nil && err.Error() != "already up-to-date" {
		return cli.NewExitError("... ["+color.RedString("FAIL")+"]", 1)
	}

	workdir, _ := repo.Worktree()
	ref, err := repo.Reference("refs/remotes/"+git.DefaultRemoteName+"/master", true)
	if err != nil {
		return cli.NewExitError("... ["+color.RedString("FAIL")+"]", 1)
	}

	head, _ := repo.Head()
	if head.Hash() == ref.Hash() {
		fmt.Println("... [" + color.CyanString("OK") + "]")
		color.Cyan("command \"%s\" already up-to-date", cmd)
		return nil
	}

	err = workdir.Checkout(&git.CheckoutOptions{
		Branch: ref.Name(),
		Force:  true,
	})

	if err != nil {
		return cli.NewExitError("... ["+color.RedString("FAIL")+"]", 1)
	}

	fmt.Println("... [" + color.GreenString("OK") + "]")

	if !installPackage(repoDir) {
		fmt.Print("Removing command...")
		if err := os.RemoveAll(repoDir); err != nil {
			return cli.NewExitError("... ["+color.RedString("FAIL")+"]", 1)
		}
		fmt.Println("... [" + color.GreenString("OK") + "]")
		return nil
	}

	return nil
}

func githubize(repo string) string {
	if strings.HasPrefix(repo, "http") || strings.HasPrefix(repo, "ssh") || strings.HasSuffix(repo, ".git") {
		return repo
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

func listDiff(oldcmds []commandPackage) {
	color.Yellow("\nAvailable Commands:")
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

	for _, newCmd := range new {
		found := false
		for _, oldCmd := range old {
			if newCmd.Name == oldCmd.Name {
				found = true
				fmt.Println("  " + newCmd.Name + "\t" + newCmd.Description)
				break
			}
		}

		if !found {
			fmt.Println(color.GreenString("  "+newCmd.Name) + "\t" + newCmd.Description)
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
			fmt.Println(color.RedString("  "+oldCmd.Name) + "\t" + oldCmd.Description)
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
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
	Arguments   string `json:"arguments"`
	Bin         string `json:"bin"`
	BinSuffix   string `json:"-"`
	OS          string `json:"-"`
	Arch        string `json:"-"`
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

func installPackage(dir string) bool {
	fmt.Print("Installing...")

	cmdPackage, err := readPackage(dir)

	if err != nil {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		fmt.Println(err.Error())
		return false
	}

	lang := determineCommandLanguage(cmdPackage)
	if lang == "" {
		fmt.Println("... [" + color.BlueString("OK") + "]")
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
					fmt.Println("... [" + color.RedString("FAIL") + "]")
					color.Red(err.Error())
					fmt.Print("Binary command(s) found, would you like to try download and install it? (Y/n): ")
					answer := ""
					fmt.Scanln(&answer)
					if answer != "" && strings.ToLower(answer) != "y" {
						return false
					}

					os.MkdirAll(dir+string(os.PathSeparator)+"bin", 0775)
				}

				fmt.Print("Downloading binary...")
				if !downloadBin(dir+string(os.PathSeparator)+"bin", cmd) {
					fmt.Println("... [" + color.RedString("FAIL") + "]")
					color.Red("Unable to download binary")
					return false
				}
				success = true
				err = nil
			}
		}
	}

	if err != nil {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		color.Red(err.Error())
		return false
	}

	if success {
		fmt.Println("... [" + color.GreenString("OK") + "]")
		return true
	}

	fmt.Println("... [" + color.CyanString("OK") + "]")
	return true
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

		if !versionCompare(matches[1], cmdPackage.Requirements.Php) {
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
		if !versionCompare(matches[1], cmdPackage.Requirements.Node) {
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
		if !versionCompare(matches[1], cmdPackage.Requirements.Ruby) {
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
	bin, err := exec.LookPath("python")
	if err != nil {
		return false, cli.NewExitError(("Unable to locate Python runtime"), 1)
	}

	if cmdPackage.Requirements.Python != "" && cmdPackage.Requirements.Python != "*" {
		cmd := exec.Command(bin, "--version")
		output, _ := cmd.CombinedOutput()
		r, _ := regexp.Compile(`Python (\d+\.\d+\.\d+).*`)
		matches := r.FindStringSubmatch(string(output))
		if !versionCompare(matches[1], cmdPackage.Requirements.Python) {
			return false, cli.NewExitError(fmt.Sprintf("Python %s is required to install this command.", cmdPackage.Requirements.Python), 1)
		}
	}

	if _, err := os.Stat(dir + string(os.PathSeparator) + "/requirements.txt"); err == nil {
		bin, err := exec.LookPath("pip")
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

	return false, cli.NewExitError("Unable to find package manager.", 1)
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
		pythonPath = pythonPaths[0]
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
		if !versionCompare(matches[1], cmdPackage.Requirements.Go) {
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

func versionCompare(compareTo string, isNewer string) bool {
	leftParts := strings.Split(compareTo, ".")
	leftMajor, _ := strconv.Atoi(leftParts[0])
	leftMinor := 0
	leftMicro := 0

	if len(leftParts) > 1 {
		leftMinor, _ = strconv.Atoi(leftParts[1])
	}
	if len(leftParts) > 2 {
		leftMicro, _ = strconv.Atoi(leftParts[2])
	}

	rightParts := strings.Split(isNewer, ".")
	rightMajor, _ := strconv.Atoi(rightParts[0])
	rightMinor := 0
	rightMicro := 0

	if len(rightParts) > 1 {
		rightMinor, _ = strconv.Atoi(rightParts[1])
	}
	if len(rightParts) > 2 {
		rightMicro, _ = strconv.Atoi(rightParts[2])
	}

	if leftMajor < rightMajor {
		return false
	}

	if leftMajor == rightMajor && leftMinor < rightMinor {
		return false
	}

	if leftMajor == rightMajor && leftMinor == rightMinor && leftMicro < rightMicro {
		return false
	}

	return true
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

func setCliTemplates() {
	cli.AppHelpTemplate = "" +
		color.YellowString("Usage: \n") +
		color.BlueString("	 {{if .UsageText}}"+
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
		`   {{join .Names ", "}}{{"\n"}}` +
		"{{end}}" +
		"{{end}}" +
		"\n{{end}}" +

		"{{if .VisibleFlags}}" +
		color.YellowString("Global Flags:\n") +
		"{{range $index, $option := .VisibleFlags}}" +
		"{{if $index}}\n{{end}}" +
		"   {{$option}}" +
		"{{end}}" +
		"\n\n{{end}}" +

		"{{if len .Authors}}" +
		color.YellowString("Author{{with $length := len .Authors}}{{if ne 1 $length}}s{{end}}{{end}}:\n") +
		"{{range $index, $author := .Authors}}{{if $index}}\n{{end}}" +
		"   {{$author}}" +
		"{{end}}" +
		"\n\n{{end}}" +

		"{{if .Copyright}}" +
		color.YellowString("Copyright:\n") +
		"   {{.Copyright}}" +
		"{{end}}\n"

	cli.CommandHelpTemplate = "" +
		color.YellowString("Name: \n") +
		"   {{.HelpName}} - {{.Usage}}\n\n" +

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
		"{{range .VisibleFlags}}   {{.}}\n{{end}}{{end}}"

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
