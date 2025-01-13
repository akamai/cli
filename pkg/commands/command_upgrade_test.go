package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/context"
)

func mockIsTTY(term *terminal.Mock, isTTY bool) {
	term.On("IsTTY").Return(isTTY).Once()
}

func mockGetAndUpdateLastUpgradeCheck(cfg *config.Mock, lastUpgradeCheckValue string) {
	cfg.On("GetValue", "cli", "last-upgrade-check").Return(lastUpgradeCheckValue, true).Once()
	cfg.On("SetValue", "cli", "last-upgrade-check", mock.AnythingOfType("string")).Return().Once()
	cfg.On("Save").Return(nil).Once()
}

func mockStartSpinner(term *terminal.Mock, msg string) {
	term.On("Spinner").Return(term).Once()
	term.On("Start", msg, []interface{}(nil)).Return().Once()
}

func mockStopSpinner(term *terminal.Mock) {
	term.On("Spinner").Return(term).Once()
	term.On("Stop", terminal.SpinnerStatusOK).Return().Once()
}

func mockFailSpinner(term *terminal.Mock) {
	term.On("Spinner").Return(term).Once()
	term.On("Fail").Return().Once()
}

func mockConfirmUpgrade(term *terminal.Mock, latestVersion, currentVersion string, userConfirms bool) {
	msg := "You can find more details about the new version here: https://github.com/akamai/cli/releases"
	term.On("Writeln", []interface{}{msg}).Return(0, nil).Once()
	question := fmt.Sprintf("New update found: %s. You are running: %s. Upgrade now?", latestVersion, currentVersion)
	term.On("Confirm", question, true).Return(userConfirms, nil).Once()
}

func TestCmdUpgrade(t *testing.T) {
	binURLRegexp := regexp.MustCompile(`/releases/download/\d+\.\d+\.\d+/akamai-\d+\.\d+\.\d+-[A-Za-z\d]+(\.exe)?$`)

	tests := map[string]struct {
		respLatestVersion     string
		init                  func(*mocked)
		forceErrorForURLRegex string
		forceBadSHAChecksum   bool
		forceBadNewBinary     bool
		expectUpgrade         bool
		expectedErrorRegex    string
	}{
		"upgrade if newer version is available": {
			respLatestVersion: "10.0.0",
			init: func(m *mocked) {
				mockStartSpinner(m.term, "Checking for upgrades...")
				mockIsTTY(m.term, true)
				mockStopSpinner(m.term)
				mockGetAndUpdateLastUpgradeCheck(m.cfg, "never")
				mockStopSpinner(m.term)
				mockConfirmUpgrade(m.term, "10.0.0", version.Version, true)
				mockStartSpinner(m.term, "Upgrading Akamai CLI")
				m.term.On("Spinner").Return(m.term).Once()
				m.term.On("OK").Return().Once()
			},
			expectUpgrade: true,
		},
		"do not upgrade if no newer version is available": {
			respLatestVersion: "1.0.0",
			init: func(m *mocked) {
				mockStartSpinner(m.term, "Checking for upgrades...")
				mockIsTTY(m.term, true)
				mockStopSpinner(m.term)
				mockGetAndUpdateLastUpgradeCheck(m.cfg, "never")
			},
			expectUpgrade: false,
		},
		"return error for bad latest release download url": {
			respLatestVersion:     "10.0.0",
			forceErrorForURLRegex: `/releases/download`,
			init: func(m *mocked) {
				mockStartSpinner(m.term, "Checking for upgrades...")
				mockIsTTY(m.term, true)
				mockStopSpinner(m.term)
				mockGetAndUpdateLastUpgradeCheck(m.cfg, "never")
				mockStopSpinner(m.term)
				mockConfirmUpgrade(m.term, "10.0.0", version.Version, true)
				mockStartSpinner(m.term, "Upgrading Akamai CLI")
				mockFailSpinner(m.term)
			},
			expectUpgrade:      false,
			expectedErrorRegex: `^Unable to download release: http://.+: 404 Not Found. Please try again.$`,
		},
		"return error for bad signature url": {
			respLatestVersion:     "10.0.0",
			forceErrorForURLRegex: `/releases/download.*\.sig$`,
			init: func(m *mocked) {
				mockStartSpinner(m.term, "Checking for upgrades...")
				mockIsTTY(m.term, true)
				mockStopSpinner(m.term)
				mockGetAndUpdateLastUpgradeCheck(m.cfg, "never")
				mockStopSpinner(m.term)
				mockConfirmUpgrade(m.term, "10.0.0", version.Version, true)
				mockStartSpinner(m.term, "Upgrading Akamai CLI")
				mockFailSpinner(m.term)
			},
			expectUpgrade:      false,
			expectedErrorRegex: `^Unable to retrieve signature for verification: http://.+\.sig: 404 Not Found. Please try again.$`,
		},
		"return error for failed upgrade": {
			respLatestVersion:   "10.0.0",
			forceBadSHAChecksum: true,
			init: func(m *mocked) {
				mockStartSpinner(m.term, "Checking for upgrades...")
				mockIsTTY(m.term, true)
				mockStopSpinner(m.term)
				mockGetAndUpdateLastUpgradeCheck(m.cfg, "never")
				mockStopSpinner(m.term)
				mockConfirmUpgrade(m.term, "10.0.0", version.Version, true)
				mockStartSpinner(m.term, "Upgrading Akamai CLI")
				mockFailSpinner(m.term)
			},
			expectUpgrade:      false,
			expectedErrorRegex: `^Checksums do not match: Updated file has wrong checksum.`,
		},
		"forward error from failed new binary execution": {
			respLatestVersion: "10.0.0",
			forceBadNewBinary: true,
			init: func(m *mocked) {
				mockStartSpinner(m.term, "Checking for upgrades...")
				mockIsTTY(m.term, true)
				mockStopSpinner(m.term)
				mockGetAndUpdateLastUpgradeCheck(m.cfg, "never")
				mockStopSpinner(m.term)
				mockConfirmUpgrade(m.term, "10.0.0", version.Version, true)
				mockStartSpinner(m.term, "Upgrading Akamai CLI")
				mockFailSpinner(m.term)
			},
			expectUpgrade:      false,
			expectedErrorRegex: `^$`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				url := r.URL.String()
				isBadURL, err := regexp.MatchString(test.forceErrorForURLRegex, url)
				assert.NoError(t, err)
				if test.forceErrorForURLRegex != "" && isBadURL {
					w.WriteHeader(http.StatusNotFound)
				} else if url == "/releases/latest" {
					w.Header().Set("Location", test.respLatestVersion)
					w.WriteHeader(http.StatusFound)
				} else if binURLRegexp.MatchString(url) {
					script := "#!/bin/bash\n" +
						"if [ \"$1\" == '--version' ]; then\n" +
						"  echo '10.0.0' >> testdata/version_out.txt\n" +
						"fi\n"
					if test.forceBadNewBinary {
						script = "plain text that OS cannot execute"
					}
					_, err := w.Write([]byte(script))
					require.NoError(t, err)
				} else if strings.HasSuffix(url, ".sig") {
					// a valid SHA256 checksum for the bash script
					checksum := "f5541d51e740f7d493269328a85935a09269f6ccc70043b5f04d6f3d92f08a0b"
					if test.forceBadSHAChecksum {
						// an invalid SHA256 checksum for the bash script
						checksum = "38a5dfa3ec07f08e8e1788d1d567359a7ed95b0e354953cf0222e0fea1872a7e"
					} else if test.forceBadNewBinary {
						// a valid SHA256 checksum for "plain text that OS cannot execute"
						checksum = "5ba86bd9afca33f5262a67c554997ac36606de83f71e983ad5f9fea3cf742772"
					}
					_, err := w.Write([]byte(checksum))
					require.NoError(t, err)
				} else {
					t.Fatalf("unknown URL: %s", url)
				}
			}))
			require.NoError(t, os.Setenv("CLI_REPOSITORY", srv.URL))
			m := &mocked{&terminal.Mock{}, &config.Mock{}, nil, nil, nil}
			command := &cli.Command{
				Name:   "upgrade",
				Action: cmdUpgrade,
			}
			app, ctx := setupTestApp(command, m)
			args := os.Args[0:1]
			cli.VersionFlag = &cli.BoolFlag{
				Name:   "version",
				Hidden: true,
			}
			args = append(args, "upgrade")

			test.init(m)
			versionDumpPath := "testdata/version_out.txt"
			if test.expectUpgrade {
				t.Cleanup(func() {
					err := os.Remove(versionDumpPath)
					if err != nil {
						t.Fatalf("failed to remove output file: %v", err)
					}
				})
			}
			err := app.RunContext(ctx, args)

			m.cfg.AssertExpectations(t)
			m.term.AssertExpectations(t)
			if test.expectedErrorRegex != "" {
				var exitErr cli.ExitCoder
				assert.ErrorAs(t, err, &exitErr)
				assert.Regexp(t, test.expectedErrorRegex, exitErr.Error())
				assert.Equal(t, 1, exitErr.ExitCode())
			} else {
				require.NoError(t, err)
			}

			if test.expectUpgrade {
				dumped, err := os.ReadFile(versionDumpPath)
				assert.NoError(t, err)
				assert.Equal(t, "10.0.0\n", string(dumped))
			}
		})
	}
}

type mockVersionProvider struct {
	mock.Mock
}

func (m *mockVersionProvider) getLatestReleaseVersion(ctx context.Context) string {
	args := m.Called(ctx)
	return args.String(0)
}

func (m *mockVersionProvider) getCurrentVersion() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockVersionProvider) mockVersions(latestVersion, currentVersion string) {
	m.On("getLatestReleaseVersion", mock.Anything).Return(latestVersion).Once()
	m.On("getCurrentVersion").Return(currentVersion).Once()
}

func Test_checkUpgradeVersion(t *testing.T) {
	tests := map[string]struct {
		forceCheck     bool
		init           func(*terminal.Mock, *config.Mock, *mockVersionProvider)
		expectedResult string
	}{
		"no upgrade if last upgrade check is set to ignore": {
			forceCheck: false,
			init: func(term *terminal.Mock, cfg *config.Mock, vp *mockVersionProvider) {
				mockIsTTY(term, true)
				cfg.On("GetValue", "cli", "last-upgrade-check").Return("ignore", true).Once()
			},
			expectedResult: "",
		},
		"return newer version if last upgrade check is set to never": {
			forceCheck: false,
			init: func(term *terminal.Mock, cfg *config.Mock, vp *mockVersionProvider) {
				mockIsTTY(term, true)
				mockGetAndUpdateLastUpgradeCheck(cfg, "never")
				vp.mockVersions("2.0.0", "1.2.3")
				mockStopSpinner(term)
				mockConfirmUpgrade(term, "2.0.0", "1.2.3", true)
			},
			expectedResult: "2.0.0",
		},
		"no upgrade if the user says no": {
			forceCheck: false,
			init: func(term *terminal.Mock, cfg *config.Mock, vp *mockVersionProvider) {
				mockIsTTY(term, true)
				mockGetAndUpdateLastUpgradeCheck(cfg, "never")
				vp.mockVersions("2.0.0", "1.2.3")
				mockStopSpinner(term)
				mockConfirmUpgrade(term, "2.0.0", "1.2.3", false)
			},
			expectedResult: "",
		},
		"return latest version if current version equals latest": {
			forceCheck: false,
			init: func(term *terminal.Mock, cfg *config.Mock, vp *mockVersionProvider) {
				mockIsTTY(term, true)
				mockGetAndUpdateLastUpgradeCheck(cfg, "never")
				vp.mockVersions("2.0.0", "2.0.0")
			},
			expectedResult: "2.0.0",
		},
		"no upgrade if only older versions exists": {
			forceCheck: false,
			init: func(term *terminal.Mock, cfg *config.Mock, vp *mockVersionProvider) {
				mockIsTTY(term, true)
				mockGetAndUpdateLastUpgradeCheck(cfg, "never")
				vp.mockVersions("1.0.0", "1.2.3")
			},
			expectedResult: "",
		},
		"no upgrade if only older versions exists, even if force": {
			forceCheck: true,
			init: func(term *terminal.Mock, cfg *config.Mock, vp *mockVersionProvider) {
				mockIsTTY(term, true)
				mockGetAndUpdateLastUpgradeCheck(cfg, "never")
				vp.mockVersions("1.0.0", "1.2.3")
			},
			expectedResult: "",
		},
		"return newer version if 24 hours passed from the last upgrade check": {
			forceCheck: false,
			init: func(term *terminal.Mock, cfg *config.Mock, vp *mockVersionProvider) {
				mockIsTTY(term, true)
				mockGetAndUpdateLastUpgradeCheck(cfg, "2024-12-31T11:55:26+01:00")
				vp.mockVersions("2.0.0", "1.2.3")
				mockStopSpinner(term)
				mockConfirmUpgrade(term, "2.0.0", "1.2.3", true)
			},
			expectedResult: "2.0.0",
		},
		"no upgrade if not TTY": {
			forceCheck: false,
			init: func(term *terminal.Mock, cfg *config.Mock, vp *mockVersionProvider) {
				mockIsTTY(term, false)
			},
			expectedResult: "",
		},
		"no upgrade if not TTY, even with force": {
			forceCheck: true,
			init: func(term *terminal.Mock, cfg *config.Mock, vp *mockVersionProvider) {
				mockIsTTY(term, false)
			},
			expectedResult: "",
		},
		"return newer version if force, even if last upgrade check is set to ignore": {
			forceCheck: true,
			init: func(term *terminal.Mock, cfg *config.Mock, vp *mockVersionProvider) {
				mockIsTTY(term, true)
				mockGetAndUpdateLastUpgradeCheck(cfg, "ignore")
				vp.mockVersions("2.0.0", "1.2.3")
				mockStopSpinner(term)
				mockConfirmUpgrade(term, "2.0.0", "1.2.3", true)
			},
			expectedResult: "2.0.0",
		},
		"return newer version if force, even if 24 hours did not pass from the last upgrade check": {
			forceCheck: true,
			init: func(term *terminal.Mock, cfg *config.Mock, vp *mockVersionProvider) {
				mockIsTTY(term, true)
				mockGetAndUpdateLastUpgradeCheck(cfg, "2099-01-27T11:55:26+01:00")
				vp.mockVersions("2.0.0", "1.2.3")
				mockStopSpinner(term)
				mockConfirmUpgrade(term, "2.0.0", "1.2.3", true)
			},
			expectedResult: "2.0.0",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			term := terminal.Mock{}
			cfg := config.Mock{}
			vp := mockVersionProvider{}
			test.init(&term, &cfg, &vp)

			ctx := terminal.Context(context.Background(), &term)
			ctx = config.Context(ctx, &cfg)
			returnedVersion := checkUpgradeVersion(ctx, test.forceCheck, &vp)

			assert.Equal(t, test.expectedResult, returnedVersion)
			term.AssertExpectations(t)
			cfg.AssertExpectations(t)
			vp.AssertExpectations(t)
		})
	}
}
