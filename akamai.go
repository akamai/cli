package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli"
	"gopkg.in/src-d/go-git.v4"
)

func main() {
	os.Setenv("AKAMAI_CLI", "1")

	app := cli.NewApp()
	app.Name = "akamai"
	app.Usage = "Akamai CLI"
	app.Version = "0.1.0"
	app.Copyright = "Copyright (C) Akamai Technologies, Inc"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "section",
			Usage:  "Section of the credentials file",
			Value:  "default",
			EnvVar: "AKAMAI_EDGERC_SECTION",
		},
	}

	helpInfo := getHelp()

	app.Commands = []cli.Command{
		{
			Name:      "help",
			Usage:     helpInfo["help"].shortDesc,
			ArgsUsage: helpInfo["Help"].prototype,
			Action:    cmdHelp,
		},
		{
			Name:      "list",
			Usage:     helpInfo["list"].shortDesc,
			ArgsUsage: helpInfo["list"].prototype,
			Action:    cmdList,
		},
		{
			Name:      "get",
			Usage:     helpInfo["get"].shortDesc,
			ArgsUsage: helpInfo["get"].prototype,
			Action:    cmdGet,
		},
		{
			Name:      "update",
			Usage:     helpInfo["update"].shortDesc,
			ArgsUsage: helpInfo["update"].prototype,
			Action:    cmdUpdate,
		},
	}

	for _, cmd := range getCommands() {
		if _, ok := helpInfo[cmd]; ok {
			continue
		}

		app.Commands = append(
			app.Commands,
			cli.Command{
				Name:            cmd,
				Usage:           "",
				ArgsUsage:       "",
				Action:          cmdSubcommand,
				Category:        "INSTALLED",
				SkipFlagParsing: true,
			},
		)
	}

	app.Run(os.Args)
}

func cmdList(c *cli.Context) {
	color.Yellow("\nAvailable Commands:")
	for _, cmd := range getCommands() {
		fmt.Println("  " + cmd)
	}
	fmt.Printf("\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", self()))
}

func cmdGet(c *cli.Context) error {
	if !c.Args().Present() {
		return cli.NewExitError(color.RedString("You must specify a repository URL"), 1)
	}

	repo := c.Args().First()

	path, err := homedir.Dir()
	if err != nil {
		return cli.NewExitError(color.RedString("Unable to determine home directory"), 1)
	}
	path += string(os.PathSeparator) + ".akamai-cli"
	_ = os.MkdirAll(path, 0775)

	cmds := getCommands()

	repo = githubize(repo)

	fmt.Print("Attempting to fetch command...")

	dirName := strings.TrimSuffix(filepath.Base(repo), ".git")

	_, err = git.PlainClone(path+string(os.PathSeparator)+dirName, false, &git.CloneOptions{
		URL:      repo,
		Progress: nil,
	})

	if err != nil {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		return cli.NewExitError(color.RedString("Unable to clone repository: "+err.Error()), 1)
	}

	fmt.Println("... [" + color.GreenString("OK") + "]")

	if !installDependencies(path + string(os.PathSeparator) + dirName) {
		//os.RemoveAll(path + string(os.PathSeparator) + dirName)
		return cli.NewExitError(color.RedString("command removed."), 1)
	}

	listDiff(cmds)

	return nil
}

func cmdUpdate(c *cli.Context) error {
	cmd := c.Args().First()

	if cmd == "" {
		help := getHelp()
		for _, cmd := range getCommands() {
			cmd := strings.ToLower(cmd)

			if _, ok := help[cmd]; !ok {
				if err := update(cmd); err != nil {
					return err
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

	args := os.Args[2:]
	return passthruCommand(executable, args)
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

			return passthruCommand(executable, args)
		}

		return cli.ShowCommandHelp(c, cmd)
	}

	return cli.ShowAppHelp(c)
}

func passthruCommand(executable string, args []string) error {
	subCmd := exec.Command(executable, args...)
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

type help map[string]struct {
	prototype string
	shortDesc string
	longDesc  string
}

func getHelp() help {
	return help{
		"help": {
			prototype: "[command] [sub-command]",
			shortDesc: "Displays help information",
		},
		"list": {
			prototype: "",
			shortDesc: "Displays available commands",
		},
		"get": {
			prototype: "<repository URL>",
			shortDesc: "Fetch and install a sub-command from a Git repository",
			//exampleCall: "%s get https://github.com/akamai-open/akamai-cli-property.git",
		},
		"update": {
			prototype: "[command]",
			shortDesc: "Update a sub-command. If no command is specified, all commands are updated",
			//exampleCall: "%s update property",
		},
	}
}

func getCommands() []string {
	sysPath := getSysPath()

	var commands []string
	var commandMap map[string]bool = make(map[string]bool)

	for cmd := range getHelp() {
		commandMap[cmd] = true
		commands = append(commands, cmd)
	}

	for _, dir := range filepath.SplitList(sysPath) {
		if dir == "" {
			dir = "."
		}
		cmdPath := filepath.Join(dir, "akamai*")
		matches, err := filepath.Glob(cmdPath)
		if err != nil {
			continue
		}

		for _, match := range matches {
			_, err := exec.LookPath(match)
			if err == nil {
				command := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(path.Base(match), "akamai-"), "akamai"))
				if len(command) != 0 {
					if _, ok := commandMap[command]; !ok {
						commandMap[command] = true
						commands = append(commands, command)
					}
				}
			}
		}
	}

	return commands
}

func getSysPath() string {
	sysPath := os.Getenv("PATH")

	sysPath += string(os.PathListSeparator) + "."

	homedir, err := homedir.Dir()
	if err == nil {
		akamaiCliPath := string(os.PathSeparator) + homedir + string(os.PathSeparator) + ".akamai-cli"
		paths, _ := filepath.Glob(akamaiCliPath + string(os.PathSeparator) + "*")
		for _, path := range paths {
			sysPath += string(os.PathListSeparator) + path
			sysPath += string(os.PathListSeparator) + path + string(os.PathSeparator) + "bin"
		}
		os.Setenv("PATH", sysPath)
	}
	return sysPath
}

func findExec(cmd string) (string, error) {
	getSysPath()

	cmd = strings.ToLower(cmd)
	var path string
	path, err := exec.LookPath("akamai-" + cmd)
	if err != nil {
		path, err = exec.LookPath("akamai" + strings.Title(cmd))
		if err != nil {
			return cmd, err
		}
	}

	return path, nil
}

func update(cmd string) error {
	exec, err := findExec(cmd)
	if err != nil {
		return cli.NewExitError(color.RedString("Command \"%s\" not found. Try \"%s help\".\n", cmd, self()), 1)
	}

	fmt.Printf("Attempting to update \"%s\" command...", cmd)

	repoDir := findGitRepo(filepath.Dir(exec))

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

	if !installDependencies(repoDir) {
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

	return "https://github.com/" + repo + ".git"
}

func findGitRepo(dir string) string {
	if _, err := os.Stat(dir + string(os.PathSeparator) + ".git"); err != nil {
		if os.IsNotExist(err) {
			if dir == "/" {
				return ""
			}

			return findGitRepo(filepath.Dir(dir))
		}
	}

	return dir
}

func listDiff(oldcmds []string) {
	color.Yellow("\nAvailable Commands:")
	cmds := getCommands()

	for _, cmd := range cmds {
		var found bool
		for _, oldcmd := range oldcmds {
			if oldcmd == cmd {
				found = true
				fmt.Println("  " + cmd)
				break
			}
		}
		if !found {
			color.Green("  " + cmd)
		}
	}

	for _, oldcmd := range oldcmds {
		var found bool
		for _, cmd := range cmds {
			if oldcmd == cmd {
				found = true
			}
		}

		if !found {
			color.Red("  " + oldcmd)
		}
	}

	fmt.Printf("\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", self()))
}

func installDependencies(dir string) bool {
	fmt.Print("Installing...")

	lang := determineCommandLanguage(dir)
	if lang == "" {
		fmt.Println("... [" + color.BlueString("OK") + "]")
		return true
	}

	var success bool
	var err error
	switch lang {
	case "php":
		success, err = installPHP(dir)
	case "javascript":
		success, err = installJavaScript(dir)
	case "ruby":
		success, err = installRuby(dir)
	case "python":
		success, err = installPython(dir)
	case "go":
		success, err = installGolang(dir)
	default:
		success = false
		err = nil
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

func determineCommandLanguage(dir string) string {
	if _, err := os.Stat(dir + string(os.PathSeparator) + "/composer.json"); err == nil {
		return "php"
	}

	if _, err := os.Stat(dir + string(os.PathSeparator) + "/yarn.lock"); err == nil {
		return "javascript"
	}

	if _, err := os.Stat(dir + string(os.PathSeparator) + "/package.json"); err == nil {
		return "javascript"
	}

	if _, err := os.Stat(dir + string(os.PathSeparator) + "/Gemfile"); err == nil {
		return "ruby"
	}

	if _, err := os.Stat(dir + string(os.PathSeparator) + "/glide.yaml"); err == nil {
		return "go"
	}

	if _, err := os.Stat(dir + string(os.PathSeparator) + "/requirements.txt"); err == nil {
		return "python"
	}

	return ""
}

func installPHP(dir string) (bool, error) {
	if _, err := os.Stat(dir + string(os.PathSeparator) + "/composer.json"); err == nil {
		if _, err := os.Stat(dir + string(os.PathSeparator) + "/composer.phar"); err == nil {
			bin, err := exec.LookPath("php")
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

func installJavaScript(dir string) (bool, error) {
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

func installRuby(dir string) (bool, error) {
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

func installPython(dir string) (bool, error) {
	if _, err := os.Stat(dir + string(os.PathSeparator) + "/requirements.txt"); err == nil {
		bin, err := exec.LookPath("pip")
		if err == nil {
			cmd := exec.Command(bin, "install", "-r", "requirements.txt")
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

func installGolang(dir string) (bool, error) {
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

	bin, err := exec.LookPath("go")
	if err == nil {
		execName := "akamai-" + strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(path.Base(dir), "akamai-"), "cli-"))

		cmd := exec.Command(bin, "build", "-o", execName, ".")
		cmd.Dir = dir
		err = cmd.Run()
		if err != nil {
			return false, cli.NewExitError(err.Error(), 1)
		}
	}

	return true, nil
}
