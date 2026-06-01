package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/git"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/packages"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/tools"
	"github.com/urfave/cli/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	thirdPartyDisclaimer = color.CyanString("Disclaimer: You are installing a third-party package, subject to its own terms and conditions. Akamai makes no warranty or representation with respect to the third-party package.")
)

var buildRawGitHubURL = func(owner, repo, branch string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/cli.json", owner, repo, branch)
}

func cmdInstall(git git.Repository, langManager packages.LangManager) cli.ActionFunc {
	return func(c *cli.Context) (e error) {
		start := time.Now()
		c.Context = log.WithCommandContext(c.Context, c.Command.Name)
		logger := log.FromContext(c.Context)
		logger.Debug("INSTALL START")
		defer func() {
			if e == nil {
				logger.Debug(fmt.Sprintf("INSTALL FINISH: %v", time.Since(start)))
			} else {
				var exitErr cli.ExitCoder
				if errors.As(e, &exitErr) && exitErr.ExitCode() == 0 {
					logger.Warn(fmt.Sprintf("INSTALL WARN: %v", e))
				} else {
					logger.Error(fmt.Sprintf("INSTALL ERROR: %v", e))
				}
			}
		}()
		if !c.Args().Present() {
			return cli.Exit(color.RedString("You must specify a repository URL"), 1)
		}

		oldCmds := getCommands(c)

		for _, repo := range c.Args().Slice() {
			repo = strings.TrimSpace(repo)

			if repo == "" {
				return cli.Exit(color.RedString("Repository URL cannot be empty"), 1)
			}

			if !strings.Contains(repo, "://") && !strings.HasPrefix(repo, "git@") {
				repo = tools.Githubize(repo)
			}

			subCmd, err := installPackage(c.Context, git, langManager, repo)
			if err != nil {
				logger.Error(fmt.Sprintf("Error installing package: %v", err))
				return err
			}
			c.App.Commands = append(c.App.Commands, subcommandToCliCommands(*subCmd, git, langManager)...)
			sortCommands(c.App.Commands)
		}

		packageListDiff(c, oldCmds)

		return nil
	}
}

func packageListDiff(c *cli.Context, oldcmds []subcommands) {
	cmds := getCommands(c)

	var old []command
	for _, oldcmd := range oldcmds {
		old = append(old, oldcmd.Commands...)
	}

	var newCmds []command
	for _, newcmd := range cmds {
		newCmds = append(newCmds, newcmd.Commands...)
	}

	var added = make(map[string]bool)
	var removed = make(map[string]bool)

	for _, newCmd := range newCmds {
		found := false
		for _, oldCmd := range old {
			if newCmd.Name == oldCmd.Name {
				found = true
				break
			}
		}

		if !found {
			added[newCmd.Name] = true
		}
	}

	for _, oldCmd := range old {
		found := false
		for _, newCmd := range newCmds {
			if newCmd.Name == oldCmd.Name {
				found = true
				break
			}
		}

		if !found {
			removed[oldCmd.Name] = true
		}
	}

	listInstalledCommands(c, added, removed)
}

func installPackage(ctx context.Context, gitRepo git.Repository, langManager packages.LangManager, repo string) (*subcommands, error) {
	logger := log.FromContext(ctx)
	logger.Debug(fmt.Sprintf("Installing package from repository: %s", repo))

	repo = strings.TrimSpace(repo)
	if repo == "" {
		userMessage := "The provided repository URL is empty or contains only whitespace. " +
			"Please specify a valid repository URL in a supported format " +
			"(e.g., 'https://github.com/<owner>/<repo>.git' or 'git@github.com:<owner>/<repo>')."
		logger.Error(userMessage)
		return nil, cli.Exit(color.RedString("%s", userMessage), 1)
	}

	srcPath, err := tools.GetAkamaiCliSrcPath()
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to get akamai cli source path: %v", err))
		return nil, err
	}

	owner, repoName := extractOwnerAndRepo(repo)
	if owner == "" || repoName == "" {
		msg := fmt.Sprintf("Unable to parse repository URL: %s", repo)
		logger.Error(msg)
		return nil, cli.Exit(color.RedString("Unable to install selected package"), 1)
	}

	dirName := repoName
	packageDir := filepath.Join(srcPath, dirName)

	if _, err = os.Stat(packageDir); err == nil {
		warningMsg := fmt.Sprintf("Package directory already exists (%s). To reinstall this package, first run 'akamai uninstall' command.", packageDir)
		logger.Warn(warningMsg)
		return nil, cli.Exit(color.YellowString("%s", warningMsg), 0)
	}

	term := terminal.Get(ctx)
	spin := term.Spinner()

	spin.Start("Attempting to fetch package configuration from %s...", repo)

	isOfficial := owner == "akamai" && strings.HasPrefix(repoName, "cli-")

	var cmdPackage subcommands

	if strings.HasPrefix(repo, "https://github.com/") {
		var fetchErr error
		for _, branch := range []string{"main", "master"} {
			url := buildRawGitHubURL(owner, repoName, branch)
			cmdPackage, fetchErr = readPackageFromGithub(url, dirName)
			if fetchErr == nil {
				break
			}
		}
		if fetchErr != nil {
			spin.Stop(terminal.SpinnerStatusFail)
			logger.Error(fmt.Sprintf("Failed to read package from github: %v", fetchErr))
			term.WriteError(fetchErr.Error())

			if strings.Contains(fetchErr.Error(), "404") {
				userMessage := fmt.Sprintf(`%s  

Reason:  
- The repository either does not exist, is private, or does not contain a 'cli.json' file.  

Steps to resolve this issue:  
1. Verify that the URL is correct.  
2. Ensure the repository is publicly accessible, or that you have the necessary permissions.  
3. Confirm that the repository contains the required 'cli.json' configuration file.  

For a list of supported packages, visit:  
%s  
`, repo, "https://techdocs.akamai.com/home/page/products-tools-a-z")

				logger.Error(strings.TrimSpace(userMessage))
				term.WriteError(strings.TrimSpace(userMessage))

				return nil, cli.Exit(color.RedString("%s", tools.CapitalizeFirstWord(git.ErrPackageNotAvailable.Error())), 1)
			}
			return nil, cli.Exit(color.RedString("Unable to install selected package"), 1)
		}
		spin.OK()
	}

	if strings.HasPrefix(repo, "https://github.com/") && isBinary(cmdPackage) {
		logger.Debug(fmt.Sprintf("Installing binaries for package in directory: %s", packageDir))
		ok, subCmd := installPackageBinaries(ctx, packageDir, cmdPackage, logger)
		if ok {
			return subCmd, nil
		}
		if err := os.RemoveAll(packageDir); err != nil {
			logger.Error(fmt.Sprintf("Failed to remove package directory: %v", err))
			return nil, err
		}
		logger.Debug(fmt.Sprintf("Unable to install binaries, cloning repository: %s", repo))
	}

	spin.Start("Attempting to fetch command from %s...", repo)

	if !isOfficial {
		term.Printf(color.CyanString("%s", thirdPartyDisclaimer))
	}

	err = gitRepo.Clone(ctx, packageDir, repo, false, spin)
	if err != nil {
		spin.Stop(terminal.SpinnerStatusFail)
		if err := os.RemoveAll(packageDir); err != nil {
			logger.Error(fmt.Sprintf("Failed to remove package directory: %v", err))
			return nil, err
		}

		logger.Error(cases.Title(language.Und, cases.NoLower).String(err.Error()))
		return nil, cli.Exit(color.RedString("%s", tools.CapitalizeFirstWord(err.Error())), 1)
	}
	spin.OK()

	logger.Debug(fmt.Sprintf("Installing dependencies for package in directory: %s", packageDir))

	ok, subCmd := installPackageDependencies(ctx, langManager, packageDir, logger)
	if !ok {
		logger.Error(fmt.Sprintf("Dependency installation failed, removing package directory: %s", packageDir))
		if err := os.RemoveAll(packageDir); err != nil {
			logger.Error(fmt.Sprintf("Failed to remove package directory: %v", err))
			return nil, err
		}
		return nil, cli.Exit("Unable to install selected package", 1)
	}
	logger.Debug(fmt.Sprintf("Dependencies installed successfully for package in directory: %s", packageDir))

	return subCmd, nil
}

func installPackageDependencies(ctx context.Context, langManager packages.LangManager, dir string, logger *slog.Logger) (bool, *subcommands) {
	term := terminal.Get(ctx)
	term.Spinner().Start("Installing Dependencies...")

	cmdPackage, err := readPackage(dir)
	if err != nil {
		term.Spinner().Stop(terminal.SpinnerStatusFail)
		logger.Error(fmt.Sprintf("Failed to read package: %v", err))
		term.WriteError(err.Error())
		return false, nil
	}

	var commands, ldFlags []string
	for _, cmd := range cmdPackage.Commands {
		commands = append(commands, cmd.Name)
		ldFlag := cmd.LdFlags
		if ldFlag != "" {
			ldFlag = fmt.Sprintf(ldFlag, cmd.Version)
		}
		ldFlags = append(ldFlags, ldFlag)
	}

	err = langManager.Install(ctx, dir, cmdPackage.Requirements, commands, ldFlags)
	if errors.Is(err, packages.ErrUnknownLang) {
		term.Spinner().WarnOK()
		warnMsg := "Package installed successfully, however package type is unknown, and may or may not function correctly."
		if _, err := term.Writeln(color.CyanString("%s", warnMsg)); err != nil {
			term.WriteError(err.Error())
			return false, nil
		}
		logger.Warn(warnMsg)

		return true, &cmdPackage
	}

	if err != nil {
		term.Spinner().Stop(terminal.SpinnerStatusFail)
		logger.Error(fmt.Sprintf("Failed to install dependecies: %v", err))
		term.WriteError(err.Error())
		return false, nil
	}

	term.Spinner().OK()

	return true, &cmdPackage
}

func installPackageBinaries(ctx context.Context, dir string, cmdPackage subcommands, logger *slog.Logger) (bool, *subcommands) {
	term := terminal.Get(ctx)
	spin := term.Spinner()
	spin.Start("Installing Binaries...")

	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0700); err != nil {
		spin.Stop(terminal.SpinnerStatusWarn)
		logger.Error(fmt.Sprintf("Unable to create directory %s: %v", filepath.Join(dir, "bin"), err))
		term.WriteError(err.Error())
		return false, nil
	}

	for _, cmd := range cmdPackage.Commands {
		err := downloadBin(ctx, filepath.Join(dir, "bin"), cmd)
		if err != nil {
			warnMsg := fmt.Sprintf("Unable to download binary: %v", err.Error())
			spin.Stop(terminal.SpinnerStatusWarn)
			if _, err := term.Writeln(color.YellowString("%s", warnMsg)); err != nil {
				term.WriteError(err.Error())
				return false, nil
			}
			logger.Warn(warnMsg)

			return false, nil
		}
	}

	err := os.WriteFile(filepath.Join(dir, "cli.json"), cmdPackage.raw, 0644)
	if err != nil {
		spin.Stop(terminal.SpinnerStatusWarn)
		warnMsg := "Unable to save configuration file " + err.Error()
		logger.Warn(warnMsg)
		if _, err := term.Writeln(color.YellowString("%s", warnMsg)); err != nil {
			term.WriteError(err.Error())
			return false, nil
		}
		return false, nil
	}

	spin.OK()
	logger.Debug(fmt.Sprintf("Binaries installed successfully in directory: %s", dir))

	return true, &cmdPackage
}

// extractOwnerAndRepo parses a GitHub repo URL and returns the owner and repo name.
func extractOwnerAndRepo(repoURL string) (string, string) {
	repoURL = strings.TrimSuffix(repoURL, ".git")

	// Handle SCP-style SSH: git@github.com:owner/repo
	if strings.HasPrefix(repoURL, "git@") {
		if i := strings.Index(repoURL, ":"); i != -1 {
			path := repoURL[i+1:]
			parts := strings.Split(strings.Trim(path, "/"), "/")
			if len(parts) >= 2 {
				return parts[0], parts[1]
			}
		}
		return "", ""
	}

	// Handle URL schemes (ssh://, https://, http://)
	if u, err := url.Parse(repoURL); err == nil && u.Path != "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 {
			return parts[len(parts)-2], parts[len(parts)-1]
		}
	}

	// Handle local file URLs
	if strings.HasPrefix(repoURL, "file://") {
		localPath := strings.TrimPrefix(repoURL, "file://")
		repoName := filepath.Base(localPath)
		if repoName != "" {
			return "local", repoName
		}
	}

	return "", ""
}
