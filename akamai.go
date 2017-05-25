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

	output, err := subCmd.CombinedOutput()
	fmt.Print(string(output))
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
		fmt.Printf(" > %s [command] [arguments]\n", self())
		color.Yellow("\nAvailable Commands:")
		for _, cmd := range getCommands() {
			fmt.Println("  " + cmd)
		}
		fmt.Printf("\nSee \"%s\" for details.\n", color.GreenString("%s help [command]", self()))

		args = []string{"help"}
	}

	help := getHelp()
	cmd := strings.ToLower(args[0])

	if _, ok := help[cmd]; ok {
		color.Yellow("\n%s\n", strings.Title(cmd))
		color.Yellow(strings.Repeat("-", len(cmd)) + "\n\n")
		fmt.Print(color.BlueString("Usage: "))
		fmt.Printf(help[cmd].prototype+"\n\n", self(), cmd)
		fmt.Printf(help[cmd].shortDesc+":\n\n", color.GreenString(cmd))
		color.Green("    "+help[cmd].exampleCall+"\n", self())
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
	fmt.Printf("\nSee \"%s\" for details.\n", color.GreenString("%s help [command]", self()))
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

	for cmd := range getHelp() {
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
					commands = append(commands, command)
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

	fmt.Print(color.YellowString("Attempting to fetch command: "))

	dirName := strings.TrimSuffix(filepath.Base(repo), ".git")

	_, err = git.PlainClone(path+string(os.PathSeparator)+dirName, false, &git.CloneOptions{
		URL:      repo,
		Progress: nil,
	})

	if err != nil {
		color.Red("Unable to clone repository: " + err.Error())
	}

	color.Green(" successfully installed command")
	list()
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

	fmt.Print(color.YellowString("Attempting to update \"%s\" command : ", cmd))

	repoDir := findGitRepo(filepath.Dir(exec))

	if repoDir == "" {
		color.Red("Unable to update, was it installed using " + color.CyanString("\"akamai get\"") + "?")
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
		color.Red("Unable to update \"%s\"", cmd)
		return
	}

	workdir, _ := repo.Worktree()
	ref, err := repo.Reference("refs/remotes/"+git.DefaultRemoteName+"/master", true)
	if err != nil {
		color.Red("Unable to update command \"%s\"", cmd)
		return
	}

	head, _ := repo.Head()
	if head.Hash() == ref.Hash() {
		color.Cyan("command \"%s\" already up-to-date", cmd)
		return
	}

	err = workdir.Checkout(&git.CheckoutOptions{
		Branch: ref.Name(),
		Force:  true,
	})

	if err != nil {
		color.Red("Unable to update command \"%s\"", cmd)
		return
	}

	color.Green("successfully updated \"%s\" command", cmd)
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
