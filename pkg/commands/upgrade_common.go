package commands

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"text/template"

	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/fatih/color"
	"github.com/inconshreveable/go-update"
	"github.com/urfave/cli/v2"
)

// UpgradeCli pulls from GitHub the latest released CLI binary and performs the upgrade of the current executable
func UpgradeCli(ctx context.Context, latestVersion string) bool {
	term := terminal.Get(ctx)
	logger := log.FromContext(ctx)

	term.Spinner().Start("Upgrading Akamai CLI")

	repo := "https://github.com/akamai/cli"
	if r := os.Getenv("CLI_REPOSITORY"); r != "" {
		repo = r
	}
	cmd := command{
		Version: latestVersion,
		Bin:     fmt.Sprintf("%s/releases/download/{{.Version}}/akamai-{{.Version}}-{{.OS}}{{.Arch}}{{.BinSuffix}}", repo),
		Arch:    runtime.GOARCH,
		OS:      runtime.GOOS,
	}

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

	resp, err := http.Get(buf.String())
	if err != nil || resp.StatusCode != http.StatusOK {
		term.Spinner().Fail()
		errMsg := color.RedString("Unable to download release, please try again.")
		logger.Error(errMsg)
		if _, err := term.Writeln(errMsg); err != nil {
			term.WriteError(err.Error())
		}
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error(err.Error())
		}
	}()

	shaResp, err := http.Get(fmt.Sprintf("%v%v", buf.String(), ".sig"))
	if err != nil || shaResp.StatusCode != http.StatusOK {
		term.Spinner().Fail()
		if _, err := term.Writeln(color.RedString("Unable to retrieve signature for verification, please try again.")); err != nil {
			term.WriteError(err.Error())
		}
		return false
	}
	defer func() {
		if err := shaResp.Body.Close(); err != nil {
			logger.Error(err.Error())
		}
	}()

	shaBody, err := ioutil.ReadAll(shaResp.Body)
	if err != nil {
		term.Spinner().Fail()
		if _, err := term.Writeln(color.RedString("Unable to retrieve signature for verification, please try again.")); err != nil {
			term.WriteError(err.Error())
		}
		return false
	}

	shaSum, err := hex.DecodeString(strings.TrimSpace(string(shaBody)))
	if err != nil {
		term.Spinner().Fail()
		if _, err := term.Writeln(color.RedString("Unable to retrieve signature for verification, please try again.")); err != nil {
			term.WriteError(err.Error())
		}
		return false
	}

	selfPath := os.Args[0]

	err = update.Apply(resp.Body, update.Options{TargetPath: selfPath, Checksum: shaSum})
	if err != nil {
		term.Spinner().Fail()
		if rerr := update.RollbackError(err); rerr != nil {
			if _, err := term.Writeln(color.RedString("Unable to install or rollback, please re-install.")); err != nil {
				term.WriteError(err.Error())
			}
			os.Exit(1)
			return false
		} else if strings.HasPrefix(err.Error(), "Upgrade file has wrong checksum.") {
			if _, err := term.Writeln(color.RedString(err.Error())); err != nil {
				term.WriteError(err.Error())
			}
			if _, err := term.Writeln(color.RedString("Checksums do not match, please try again.")); err != nil {
				term.WriteError(err.Error())
			}
			return false
		}
		if _, err := term.Writeln(color.RedString(err.Error())); err != nil {
			term.WriteError(err.Error())
		}
		return false
	}

	term.Spinner().OK()

	if err == nil {
		os.Args[0] = selfPath
	}

	subCmd := createCommand(os.Args[0], os.Args[1:])
	if err = passthruCommand(ctx, subCmd, packages.NewLangManager(), packages.LanguageRequirements{}, selfPath); err != nil {
		cli.OsExiter(1)
		return false
	}
	cli.OsExiter(0)

	return true
}
