package app

import (
	"fmt"
	akamai "github.com/akamai/cli-common-golang"
	"github.com/akamai/cli/pkg/commands"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/stats"
	"github.com/akamai/cli/pkg/tools"
	"github.com/akamai/cli/pkg/version"
	"github.com/kardianos/osext"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Run() int {
	os.Setenv("AKAMAI_CLI", "1")
	os.Setenv("AKAMAI_CLI_VERSION", version.Version)

	cachePath := config.GetConfigValue("cli", "cache-path")
	if cachePath == "" {
		cliHome, _ := tools.GetAkamaiCliPath()

		cachePath = filepath.Join(cliHome, "cache")
		err := os.MkdirAll(cachePath, 0700)
		if err != nil {
			return 1
		}
	}

	config.SetConfigValue("cli", "cache-path", cachePath)
	config.SaveConfig()
	config.ExportConfigEnv()
	createApp()

	log.Setup()

	if err := firstRun(); err != nil {
		return 2
	}
	checkUpgrade()
	stats.CheckPing()
	if err := akamai.App.Run(os.Args); err != nil {
		return 3
	}
	return 0
}

func createApp() {
	akamai.CreateApp("", "Akamai CLI", "", version.Version, "", commands.CommandLocator)

	akamai.App.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "bash",
			Usage: "Output bash auto-complete",
		},
		cli.BoolFlag{
			Name:  "zsh",
			Usage: "Output zsh auto-complete",
		},
		cli.StringFlag{
			Name:  "proxy",
			Usage: "Set a proxy to use",
		},
		cli.BoolFlag{
			Name:   "daemon",
			Usage:  "Keep Akamai CLI running in the background, particularly useful for Docker containers",
			Hidden: true,
			EnvVar: "AKAMAI_CLI_DAEMON",
		},
	}

	akamai.App.Action = func(c *cli.Context) {
		defaultAction(c)
	}

	akamai.App.Before = func(c *cli.Context) error {
		if c.IsSet("proxy") {
			proxy := c.String("proxy")
			os.Setenv("HTTP_PROXY", proxy)
			os.Setenv("http_proxy", proxy)
			if strings.HasPrefix(proxy, "https") {
				os.Setenv("HTTPS_PROXY", proxy)
				os.Setenv("https_proxy", proxy)
			}
		}

		if c.IsSet("daemon") {
			for {
				time.Sleep(time.Hour * 24)
			}
		}
		return nil
	}
}

func checkUpgrade() {
	if latestVersion := commands.CheckUpgradeVersion(false); latestVersion != "" {
		if commands.UpgradeCli(latestVersion) {
			stats.TrackEvent("upgrade.auto", "success", "to: "+latestVersion+" from: "+version.Version)
			return
		}
		stats.TrackEvent("upgrade.auto", "failed", "to: "+latestVersion+" from: "+version.Version)
	}
}

func defaultAction(c *cli.Context) {
	cmd, err := osext.Executable()
	if err != nil {
		cmd = tools.Self()
	}

	zshScript := `set -k
# To enable zsh auto-completion, run: eval "$(` + cmd + ` --zsh)"
# We recommend adding this to your .zshrc file
autoload -U compinit && compinit
autoload -U bashcompinit && bashcompinit`

	bashComments := `# To enable bash auto-completion, run: eval "$(` + cmd + ` --bash)"
# We recommend adding this to your .bashrc or .bash_profile file`

	bashScript := `_akamai_cli_bash_autocomplete() {
    local cur opts base
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-auto-complete )
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

complete -F _akamai_cli_bash_autocomplete ` + tools.Self()

	if c.Bool("bash") {
		fmt.Fprintln(akamai.App.Writer, bashComments)
		fmt.Fprintln(akamai.App.Writer, bashScript)
		return
	}

	if c.Bool("zsh") {
		fmt.Fprintln(akamai.App.Writer, zshScript)
		fmt.Fprintln(akamai.App.Writer, bashScript)
		return
	}

	cli.ShowAppHelpAndExit(c, 0)
}
