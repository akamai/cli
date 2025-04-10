package commands

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/packages"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/inconshreveable/go-update"
	"github.com/urfave/cli/v2"
)

// UpgradeCli pulls from GitHub the latest released CLI binary and performs the upgrade of the current executable
func UpgradeCli(ctx context.Context, latestVersion string) (e error) {
	term := terminal.Get(ctx)
	logger := log.FromContext(ctx)
	start := time.Now()

	term.Spinner().Start("Upgrading Akamai CLI")
	defer func() {
		if e == nil {
			term.Spinner().OK()
			logger.Debug(fmt.Sprintf("UPGRADE FINISH: %v", time.Since(start)))
		} else {
			term.Spinner().Fail()
			logger.Error(fmt.Sprintf("UPGRADE ERROR: %v", e))
		}
	}()

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
		return cli.Exit(color.RedString("Templating error: %s", err), 1)
	}

	resp, err := http.Get(buf.String())
	if err != nil || resp.StatusCode != http.StatusOK {
		logger.Error(fmt.Sprintf("Unable to get release: %s", err))
		var reason string
		if err == nil {
			reason = fmt.Sprintf("%s: %s", buf.String(), resp.Status)
		} else {
			reason = err.Error()
		}
		return cli.Exit(color.RedString("Unable to download release: %s. Please try again.", reason), 1)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error(err.Error())
		}
	}()

	shaURL := fmt.Sprintf("%v%v", buf.String(), ".sig")
	shaResp, err := http.Get(shaURL)
	if err != nil || shaResp.StatusCode != http.StatusOK {
		var reason string
		if err == nil {
			reason = fmt.Sprintf("%s: %s", shaURL, shaResp.Status)
		} else {
			reason = err.Error()
		}
		return cli.Exit(color.RedString("Unable to retrieve signature for verification: %s. Please try again.", reason), 1)
	}
	defer func() {
		if err := shaResp.Body.Close(); err != nil {
			logger.Error(err.Error())
		}
	}()

	shaBody, err := io.ReadAll(shaResp.Body)
	if err != nil {
		return cli.Exit(color.RedString("Unable to retrieve signature for verification: %s. Please try again.", err.Error()), 1)
	}

	shaSum, err := hex.DecodeString(strings.TrimSpace(string(shaBody)))
	if err != nil {
		return cli.Exit(color.RedString("Unable to retrieve signature for verification: %s. Please try again.", err.Error()), 1)
	}

	selfPath := os.Args[0]

	err = update.Apply(resp.Body, update.Options{TargetPath: selfPath, Checksum: shaSum})
	if err != nil {
		if rerr := update.RollbackError(err); rerr != nil {
			return cli.Exit(color.RedString("Unable to install or rollback: %s. Please re-install.", rerr.Error()), 1)
		}
		if strings.HasPrefix(err.Error(), "Updated file has wrong checksum.") {
			return cli.Exit(color.RedString("Checksums do not match: %s. Please try again.", err.Error()), 1)
		}
		return cli.Exit(color.RedString("Unable to upgrade: %s", err), 1)
	}

	os.Args[0] = selfPath
	subCmd := createCommand(os.Args[0], os.Args[1:])
	return passthruCommand(ctx, subCmd, packages.NewLangManager(), packages.LanguageRequirements{}, selfPath)
}
