package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

var (
	builtInCommands = []string{"help", "list"}
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
			fmt.Printf("Command \"%s\" not found. Try \"%s help\".\n", cmd, self)
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
	}
}

func getCommands() []string {
	sysPath := os.Getenv("PATH")
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

func findExec(cmd string) (string, error) {
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
