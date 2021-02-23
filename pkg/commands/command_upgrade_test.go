package commands

import (
	"fmt"
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"runtime"
	"testing"
)

func TestCmdUpgrade(t *testing.T) {
	binURLRegexp := regexp.MustCompile(`/archive/[0-9]+\.[0-9]+\.[0-9]+\.zip$`)
	tests := map[string]struct {
		args              []string
		respLatestVersion string
		init              func(*mocked)
		withError         string
	}{
		"last upgrade check is set to never": {
			args:              []string{"cli.testKey", "testValue"},
			respLatestVersion: "10.0.0",
			init: func(m *mocked) {

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Checking for upgrades...", []interface{}(nil)).Return().Once()

				// Checking if cli should be upgraded
				m.term.On("IsTTY").Return(true).Once()
				m.cfg.On("GetValue", "cli", "last-upgrade-check").Return("never", true).Once()
				m.cfg.On("SetValue", "cli", "last-upgrade-check", mock.AnythingOfType("string")).Return().Once()
				m.cfg.On("Save").Return(nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusOK).Return().Once()
				m.term.On("Confirm", fmt.Sprintf("New upgrade found: 10.0.0 (you are running: %s). Upgrade now? [Y/n]: ", version.Version), true).Return(true, nil).Once()

				// start upgrade
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Upgrading Akamai CLI", []interface{}(nil)).Return().Once()
				m.langManager.On("Install", "cli-upgrade-10.0.0/cli-10.0.0", packages.LanguageRequirements{Go: runtime.Version()}, []string{"cli/main.go"}).
					Return(nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
			},
		},
		"24 hours passed, upgrade": {
			args:              []string{"cli.testKey", "testValue"},
			respLatestVersion: "10.0.0",
			init: func(m *mocked) {

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Checking for upgrades...", []interface{}(nil)).Return().Once()

				// Checking if cli should be upgraded
				m.term.On("IsTTY").Return(true).Once()
				m.cfg.On("GetValue", "cli", "last-upgrade-check").Return("2021-02-10T11:55:26+01:00", true).Once()
				m.cfg.On("SetValue", "cli", "last-upgrade-check", mock.AnythingOfType("string")).Return().Once()
				m.cfg.On("Save").Return(nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Stop", terminal.SpinnerStatusOK).Return().Once()
				m.term.On("Confirm", fmt.Sprintf("New upgrade found: 10.0.0 (you are running: %s). Upgrade now? [Y/n]: ", version.Version), true).Return(true, nil).Once()

				// start upgrade
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Upgrading Akamai CLI", []interface{}(nil)).Return().Once()
				m.langManager.On("Install", "cli-upgrade-10.0.0/cli-10.0.0", packages.LanguageRequirements{Go: runtime.Version()}, []string{"cli/main.go"}).
					Return(nil).Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				url := r.URL.String()
				if url == "/releases/latest" {
					w.Header().Set("Location", test.respLatestVersion)
					w.WriteHeader(http.StatusFound)
				} else if binURLRegexp.MatchString(url) {
					resp, err := ioutil.ReadFile("testdata/cli-upgrade/cli-10.0.0.zip")
					require.NoError(t, err)
					_, err = w.Write(resp)
					require.NoError(t, err)
				} else {
					t.Fatalf("unknown URL: %s", url)
				}
			}))
			require.NoError(t, os.Setenv("CLI_REPOSITORY", srv.URL))
			m := &mocked{&terminal.Mock{}, &config.Mock{}, nil, &packages.Mock{}}
			command := &cli.Command{
				Name:   "upgrade",
				Action: cmdUpgrade(m.langManager),
			}
			app, ctx := setupTestApp(command, m)
			cli.OsExiter = func(rc int) {}
			args := os.Args[0:1]
			cli.VersionFlag = &cli.BoolFlag{
				Name:   "version",
				Hidden: true,
			}
			args = append(args, "upgrade")
			args = append(args, test.args...)

			test.init(m)
			err := app.RunContext(ctx, args)

			m.cfg.AssertExpectations(t)
			if test.withError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.withError)
				return
			}
			require.NoError(t, err)
		})
	}
}
