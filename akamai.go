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
	"gopkg.in/src-d/go-git.v4"
)

func main() {
	self := self()
	cmd := "help"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch {
	case cmd == "help":
		if len(os.Args) > 2 {
			help(os.Args[2:])
		} else {
			help([]string{})
		}
		return
	case cmd == "list":
		list()
		return
	case cmd == "get":
		if len(os.Args) < 3 {
			color.Red("You must specify a repository URL")
			help([]string{"get"})
			return
		}
		get(os.Args[2])
		return
	case cmd == "update":
		if len(os.Args) < 3 {
			update("")
			return
		}
		update(os.Args[2])
		return
	}

	executable, err := findExec(cmd)

	if err != nil {
		fmt.Printf("Command \"%s\" not found. Try \"%s help\".\n", cmd, self)
		return
	}
	args := os.Args[2:]

	subCmd := exec.Command(executable, args...)
	subCmd.Stdin = os.Stdin
	subCmd.Stderr = os.Stderr
	subCmd.Stdout = os.Stdout

	err = subCmd.Run()
	//fmt.Print(string(output))
	if err != nil {
		os.Exit(1)
	}
}

func self() string {
	return path.Base(os.Args[0])
}

func help(args []string) {
	if len(args) == 0 {
		color.Yellow("Usage:")
		color.Blue("  %s [command] [arguments]\n", self())
		color.Yellow("\nAvailable Commands:")
		for _, cmd := range getCommands() {
			fmt.Println("  " + cmd)
		}
		fmt.Printf("\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", self()))

		args = []string{"help"}
	}

	help := getHelp()
	cmd := strings.ToLower(args[0])

	if _, ok := help[cmd]; ok {
		color.Yellow("\n%s\n", strings.Title(cmd))
		color.Yellow(strings.Repeat("-", len(cmd)) + "\n\n")
		fmt.Print("Usage: ")
		color.Blue(help[cmd].prototype+"\n\n", self(), cmd)
		fmt.Printf(help[cmd].shortDesc+":\n\n", color.BlueString(cmd))
		color.Blue("    "+help[cmd].exampleCall+"\n", self())
	} else {
		executable, err := findExec(cmd)

		if err != nil {
			fmt.Printf("Command \"%s\" not found. Try \"%s help\".\n", cmd, self())
			return
		}
		args := []string{"help"}
		if len(os.Args) > 2 {
			args = append(args, os.Args[3:]...)
		}

		subCmd := exec.Command(executable, args...)
		output, err := subCmd.Output()
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Print(string(output))
	}

}

func list() {
	color.Yellow("\nAvailable Commands:")
	for _, cmd := range getCommands() {
		fmt.Println("  " + cmd)
	}
	fmt.Printf("\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", self()))
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

type Help map[string]struct {
	prototype   string
	shortDesc   string
	longDesc    string
	exampleCall string
}

func getHelp() Help {
	return Help{
		"help": {
			prototype:   "%s %s [command]",
			shortDesc:   "The %s command displays help for a given command",
			exampleCall: "%s help list",
		},
		"list": {
			prototype:   "%s %s",
			shortDesc:   "The %s command displays available commands",
			exampleCall: "%s list",
		},
		"get": {
			prototype:   "%s %s <repository URL>",
			shortDesc:   "The %s command will fetch and install a sub-command from a Git repository",
			exampleCall: "%s get https://github.com/akamai-open/akamai-cli-property.git",
		},
		"update": {
			prototype:   "%s %s [command]",
			shortDesc:   "The %s command will update a sub-command. If no command is specified, all commands are updated",
			exampleCall: "%s update property",
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

func get(repo string) {
	path, err := homedir.Dir()
	if err != nil {
		fmt.Println("Unable to determine home directory")
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
		color.Red("Unable to clone repository: " + err.Error())
	}

	fmt.Println("... [" + color.GreenString("OK") + "]")

	if !installDependencies(path + string(os.PathSeparator) + dirName) {
		os.RemoveAll(path + string(os.PathSeparator) + dirName)
		color.Red("command removed.")
		return
	}

	listDiff(cmds)
}

func githubize(repo string) string {
	if strings.HasPrefix(repo, "http") || strings.HasPrefix(repo, "ssh") || strings.HasSuffix(repo, ".git") {
		return repo
	}

	return "https://github.com/" + repo + ".git"
}

func update(cmd string) {
	if cmd == "" {
		help := getHelp()
		for _, cmd := range getCommands() {
			cmd := strings.ToLower(cmd)

			if _, ok := help[cmd]; !ok {
				update(cmd)
			}
		}

		return
	}

	exec, err := findExec(cmd)
	if err != nil {
		color.Red(err.Error())
		fmt.Printf("Command \"%s\" not found. Try \"%s help\".\n", cmd, self())
		return
	}

	fmt.Printf("Attempting to update \"%s\" command...", cmd)

	repoDir := findGitRepo(filepath.Dir(exec))

	if repoDir == "" {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		color.Red("unable to update, was it installed using " + color.CyanString("\"akamai get\"") + "?")
		return
	}

	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		color.Red(err.Error())
		return
	}

	err = repo.Fetch(&git.FetchOptions{
		RemoteName: git.DefaultRemoteName,
	})

	if err != nil && err.Error() != "already up-to-date" {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		return
	}

	workdir, _ := repo.Worktree()
	ref, err := repo.Reference("refs/remotes/"+git.DefaultRemoteName+"/master", true)
	if err != nil {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		return
	}

	head, _ := repo.Head()
	if head.Hash() == ref.Hash() {
		fmt.Println("... [" + color.CyanString("OK") + "]")
		color.Cyan("command \"%s\" already up-to-date", cmd)
		return
	}

	err = workdir.Checkout(&git.CheckoutOptions{
		Branch: ref.Name(),
		Force:  true,
	})

	if err != nil {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		return
	}

	fmt.Println("... [" + color.GreenString("OK") + "]")

	if !installDependencies(repoDir) {
		fmt.Print("Removing command...")
		if err := os.RemoveAll(repoDir); err != nil {
			fmt.Println("... [" + color.RedString("FAIL") + "]")
			return
		}
		fmt.Println("... [" + color.GreenString("OK") + "]")
		return
	}
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

func installDependencies(dir string) bool {
	fmt.Print("Installing Dependencies...")

	success, err := installPHPDeps(dir)
	if err != nil {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		return false
	}

	if success {
		fmt.Println("... [" + color.GreenString("OK") + "]")
		return true
	}

	success, err = installJavaScriptDeps(dir)
	if err != nil {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		return false
	}

	if success {
		fmt.Println("... [" + color.GreenString("OK") + "]")
		return true
	}

	if !success {
		fmt.Println("... [" + color.RedString("FAIL") + "]")
		return false
	}

	fmt.Println("... [" + color.CyanString("OK") + "]")
	return true
}

func installPHPDeps(dir string) (bool, error) {
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

		return true, nil
	}

	return false, nil
}

func installJavaScriptDeps(dir string) (bool, error) {
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
		// Specifically _not_ returning here as NPM can install from the same package.json
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

	return false, nil
}
