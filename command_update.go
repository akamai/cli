package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli"
	"gopkg.in/src-d/go-git.v4"
)

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