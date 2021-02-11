package commands

import (
	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/version"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestCmdUpgrade(t *testing.T) {
	binUrlRx := regexp.MustCompile(`/releases/download/[0-9]+\.[0-9]+\.[0-9]+/akamai-[0-9]+\.[0-9]+\.[0-9]+-[A-Za-z0-9]+$`)
	tests := map[string]struct {
		args              []string
		respLatestVersion string
		init              func(*mocked)
		expectedExitCode  int
		withError         string
	}{
		//"set config no error": {
		//	args:              []string{"cli.testKey", "testValue"},
		//	respLatestVersion: "10.0.0",
		//	init: func(m *mocked) {
		//
		//		m.term.On("Spinner").Return(m.term).Once()
		//		m.term.On("Start", "Checking for upgrades...", []interface{}(nil)).Return().Once()
		//
		//		// Checking if cli should be upgraded
		//		m.term.On("IsTTY").Return(true).Once()
		//		m.cfg.On("GetValue", "cli", "last-upgrade-check").Return("never", true).Once()
		//		m.cfg.On("SetValue", "cli", "last-upgrade-check", mock.AnythingOfType("string")).Return().Once()
		//		m.cfg.On("Save").Return(nil).Once()
		//
		//		m.term.On("Spinner").Return(m.term).Once()
		//		m.term.On("Stop", terminal.SpinnerStatusOK).Return().Once()
		//		m.term.On("Printf", "Found new version: %s (current version: %s)\n", []interface{}{color.BlueString("v10.0.0"), color.BlueString("v" + version.Version)})
		//
		//		// start upgrade
		//		m.term.On("Spinner").Return(m.term).Once()
		//		m.term.On("Start", "Upgrading Akamai CLI", []interface{}(nil)).Return().Once()
		//
		//		m.term.On("Spinner").Return(m.term).Once()
		//		m.term.On("OK").Return().Once()
		//
		//		m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
		//	},
		//	expectedExitCode: 1,
		//},
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
				m.term.On("Printf", "Found new version: %s (current version: %s)\n", []interface{}{color.BlueString("v10.0.0"), color.BlueString("v" + version.Version)})

				// start upgrade
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("Start", "Upgrading Akamai CLI", []interface{}(nil)).Return().Once()

				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()

				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true)
			},
			expectedExitCode: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				url := r.URL.String()
				if url == "/releases/latest" {
					w.Header().Set("Location", test.respLatestVersion)
					w.WriteHeader(http.StatusFound)
				} else if binUrlRx.MatchString(url) {
					_, err := w.Write([]byte("binary file"))
					require.NoError(t, err)
				} else if strings.HasSuffix(url, ".sig") {
					// a valid SHA256 checksum for "binary file" string
					_, err := w.Write([]byte("9a3924b98ad3ce5e51d2c84a7129054c2523f39643a6ea27f8118511ecd4cdba"))
					require.NoError(t, err)
				} else {
					t.Fatalf("unknown URL: %s", url)
				}
			}))
			require.NoError(t, os.Setenv("CLI_REPOSITORY", srv.URL))
			m := &mocked{&terminal.Mock{}, &config.Mock{}, nil, nil}
			command := &cli.Command{
				Name:   "upgrade",
				Action: cmdUpgrade,
			}
			app, ctx := setupTestApp(command, m)
			cli.OsExiter = func(rc int) {
				assert.Equal(t, test.expectedExitCode, rc)
			}
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
