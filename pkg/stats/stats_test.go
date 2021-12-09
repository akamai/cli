package stats

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/akamai/cli/pkg/config"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/akamai/cli/pkg/version"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mocked struct {
	cfg  *config.Mock
	term *terminal.Mock
}

func TestTrackEvent(t *testing.T) {
	tests := map[string]struct {
		givenCategory string
		givenAction   string
		givenValue    string
		withDebug     bool
		init          func(*mocked)
		expectedURL   string
		expectedBody  string
	}{
		"send stats, no debug, with client ID": {
			givenCategory: "test-category",
			givenAction:   "test-action",
			givenValue:    "test-value",
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "client-id").Return("123", true).Once()
			},
			expectedURL:  "/collect",
			expectedBody: `aip=1&cid=123&ea=test-action&ec=test-category&el=test-value&t=event&tid=UA-34796267-23&v=1`,
		},
		"send stats, with debug, with client ID": {
			givenCategory: "test-category",
			givenAction:   "test-action",
			givenValue:    "test-value",
			withDebug:     true,
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "client-id").Return("123", true).Once()
				m.term.On("Writeln", []interface{}{"stats uploaded"}).Return(0, nil).Once()
			},
			expectedURL:  "/debug/collect",
			expectedBody: `aip=1&cid=123&ea=test-action&ec=test-category&el=test-value&t=event&tid=UA-34796267-23&v=1`,
		},
		"stats disabled": {
			givenCategory: "test-category",
			givenAction:   "test-action",
			givenValue:    "test-value",
			withDebug:     true,
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true).Once()
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, test.expectedURL, r.URL.String())
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
				body, err := ioutil.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, test.expectedBody, string(body))
				_, err = w.Write([]byte(`stats uploaded`))
				assert.NoError(t, err)
			}))
			require.NoError(t, os.Setenv("AKAMAI_CLI_ANALYTICS_URL", srv.URL))
			if test.withDebug {
				require.NoError(t, os.Setenv("AKAMAI_CLI_DEBUG_ANALYTICS", "true"))
				defer func() {
					require.NoError(t, os.Unsetenv("AKAMAI_CLI_DEBUG_ANALYTICS"))
				}()
			}
			m := &mocked{&config.Mock{}, &terminal.Mock{}}
			ctx := terminal.Context(context.Background(), m.term)
			ctx = config.Context(ctx, m.cfg)
			test.init(m)

			TrackEvent(ctx, test.givenCategory, test.givenAction, test.givenValue)
			m.cfg.AssertExpectations(t)
			m.term.AssertExpectations(t)
		})
	}
}

func TestCheckPing(t *testing.T) {
	tests := map[string]struct {
		init      func(*mocked)
		withError bool
	}{
		"ping never check, do check": {
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "last-ping").Return("never", true).Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "client-id").Return("123", true).Once()
				m.cfg.On("SetValue", "cli", "last-ping", mock.AnythingOfType("string")).Return().Once()
				m.cfg.On("Save").Return(nil).Once()
			},
		},
		"more that 24 hours passed, check ping": {
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "last-ping").Return("2021-02-10T11:55:26+01:00", true).Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "client-id").Return("123", true).Once()
				m.cfg.On("SetValue", "cli", "last-ping", mock.AnythingOfType("string")).Return().Once()
				m.cfg.On("Save").Return(nil).Once()
			},
		},
		"stats disabled": {
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("false", true).Once()
			},
		},
		"date parsing error": {
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "last-ping").Return("123", true).Once()
			},
			withError: true,
		},
		"error on save": {
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "last-ping").Return("never", true).Once()
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "client-id").Return("123", true).Once()
				m.cfg.On("SetValue", "cli", "last-ping", mock.AnythingOfType("string")).Return().Once()
				m.cfg.On("Save").Return(fmt.Errorf("oops")).Once()
			},
			withError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/collect", r.URL.String())
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
				body, err := ioutil.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, `aip=1&cid=123&ea=daily&ec=ping&el=pong&t=event&tid=UA-34796267-23&v=1`, string(body))
				_, err = w.Write([]byte(`stats uploaded`))
				assert.NoError(t, err)
			}))
			require.NoError(t, os.Setenv("AKAMAI_CLI_ANALYTICS_URL", srv.URL))
			m := &mocked{&config.Mock{}, &terminal.Mock{}}
			ctx := terminal.Context(context.Background(), m.term)
			ctx = config.Context(ctx, m.cfg)
			test.init(m)

			err := CheckPing(ctx)
			m.cfg.AssertExpectations(t)
			if test.withError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestFirstRunCheckStats(t *testing.T) {
	tests := map[string]struct {
		bannerShown  bool
		init         func(*mocked)
		expectedURL  string
		expectedBody string
	}{
		"show banner, enable stats": {
			bannerShown: false,
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("", false).Once()

				mockShowBanner(m.term)

				anonymous := color.New(color.FgWhite, color.Bold).Sprint("anonymous")
				m.term.On("Printf", "Help Akamai improve Akamai CLI by automatically sending %s diagnostics and usage data.\n",
					[]interface{}{anonymous}).Return().Once()
				m.term.On("Writeln",
					[]interface{}{"Examples of data being sent include upgrade statistics, and packages installed and updated."}).
					Return(0, nil).Once()
				m.term.On("Writeln",
					[]interface{}{"Note: if you choose to opt-out, a single %s event will be submitted to help track overall usage.\n", anonymous}).
					Return(0, nil).Once()
				m.term.On("Confirm", fmt.Sprintf("Send %s diagnostics and usage data to Akamai? [Y/n]: ", anonymous), true).
					Return(true, nil).Once()

				m.cfg.On("SetValue", "cli", "enable-cli-statistics", statsVersion).Return().Once()
				m.cfg.On("SetValue", "cli", "stats-version", statsVersion).Return().Once()
				m.cfg.On("SetValue", "cli", "last-ping", "never").Return().Once()
				m.cfg.On("GetValue", "cli", "client-id").Return("", false).Once()
				m.cfg.On("SetValue", "cli", "client-id", mock.AnythingOfType("string")).Return().Once()
				m.cfg.On("Save").Return(nil).Once()

				// track "first-run" event
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "client-id").Return("123", true).Once()
			},
			expectedBody: `aip=1&cid=123&ea=stats-enabled&ec=first-run&el=1.1&t=event&tid=UA-34796267-23&v=1`,
		},
		"show banner, disable stats": {
			bannerShown: false,
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("", false).Once()

				mockShowBanner(m.term)

				anonymous := color.New(color.FgWhite, color.Bold).Sprint("anonymous")
				m.term.On("Printf", "Help Akamai improve Akamai CLI by automatically sending %s diagnostics and usage data.\n",
					[]interface{}{anonymous}).Return().Once()
				m.term.On("Writeln",
					[]interface{}{"Examples of data being sent include upgrade statistics, and packages installed and updated."}).
					Return(0, nil).Once()
				m.term.On("Writeln",
					[]interface{}{"Note: if you choose to opt-out, a single %s event will be submitted to help track overall usage.\n", anonymous}).
					Return(0, nil).Once()
				m.term.On("Confirm", fmt.Sprintf("Send %s diagnostics and usage data to Akamai? [Y/n]: ", anonymous), true).
					Return(false, nil).Once()

				// track "opt-out" event
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "client-id").Return("123", true).Once()

				m.cfg.On("SetValue", "cli", "enable-cli-statistics", "false").Return().Once()
				m.cfg.On("Save").Return(nil).Once()
			},
			expectedBody: `aip=1&cid=123&ea=stats-opt-out&ec=first-run&el=true&t=event&tid=UA-34796267-23&v=1`,
		},
		"migrate stats": {
			bannerShown: false,
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("abc", true).Once()

				mockShowBanner(m.term)

				// migrate stats
				m.cfg.On("GetValue", "cli", "stats-version").Return("1.0", true).Once()

				anonymous := color.New(color.FgWhite, color.Bold).Sprint("anonymous")
				m.term.On("Printf", "Akamai CLI has changed the %s data it collects. It now additionally collects the following: \n\n",
					[]interface{}{anonymous}).Return().Once()
				m.term.On("Printf", " - %s\n", []interface{}{"command name executed (no arguments)"}).Return().Once()
				m.term.On("Printf", " - %s\n", []interface{}{"command version executed"}).Return().Once()
				m.term.On("Printf", "\nTo continue collecting %s statistics, Akamai CLI requires that you re-affirm you decision.\n",
					[]interface{}{anonymous}).Return().Once()
				m.term.On("Writeln",
					[]interface{}{"Note: if you choose to opt-out, a single anonymous event will be submitted to help track overall usage.\n"}).
					Return(0, nil).Once()
				m.term.On("Confirm", fmt.Sprintf("Continue sending %s diagnostics and usage data to Akamai? [Y/n]: ", anonymous), true).
					Return(true, nil).Once()
				m.cfg.On("SetValue", "cli", "stats-version", statsVersion).Return().Once()
				m.cfg.On("Save").Return(nil).Once()

				// track "stats-update-opt-in" event
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "client-id").Return("123", true).Once()
			},
			expectedBody: `aip=1&cid=123&ea=stats-update-opt-in&ec=first-run&el=1.1&t=event&tid=UA-34796267-23&v=1`,
		},
		"migrate stats, opt-out": {
			bannerShown: false,
			init: func(m *mocked) {
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("abc", true).Once()

				mockShowBanner(m.term)

				// migrate stats
				m.cfg.On("GetValue", "cli", "stats-version").Return("1.0", true).Once()

				anonymous := color.New(color.FgWhite, color.Bold).Sprint("anonymous")
				m.term.On("Printf", "Akamai CLI has changed the %s data it collects. It now additionally collects the following: \n\n",
					[]interface{}{anonymous}).Return().Once()
				m.term.On("Printf", " - %s\n", []interface{}{"command name executed (no arguments)"}).Return().Once()
				m.term.On("Printf", " - %s\n", []interface{}{"command version executed"}).Return().Once()
				m.term.On("Printf", "\nTo continue collecting %s statistics, Akamai CLI requires that you re-affirm you decision.\n",
					[]interface{}{anonymous}).Return().Once()
				m.term.On("Writeln",
					[]interface{}{"Note: if you choose to opt-out, a single anonymous event will be submitted to help track overall usage.\n"}).
					Return(0, nil).Once()
				m.term.On("Confirm", fmt.Sprintf("Continue sending %s diagnostics and usage data to Akamai? [Y/n]: ", anonymous), true).
					Return(false, nil).Once()
				m.cfg.On("SetValue", "cli", "enable-cli-statistics", "false").Return().Once()
				m.cfg.On("Save").Return(nil).Once()

				// track "stats-update-opt-in" event
				m.cfg.On("GetValue", "cli", "enable-cli-statistics").Return("true", true).Once()
				m.cfg.On("GetValue", "cli", "client-id").Return("123", true).Once()
			},
			expectedBody: `aip=1&cid=123&ea=stats-update-opt-out&ec=first-run&el=1.1&t=event&tid=UA-34796267-23&v=1`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/collect", r.URL.String())
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
				body, err := ioutil.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, test.expectedBody, string(body))
				_, err = w.Write([]byte(`stats uploaded`))
				assert.NoError(t, err)
			}))
			require.NoError(t, os.Setenv("AKAMAI_CLI_ANALYTICS_URL", srv.URL))
			m := &mocked{&config.Mock{}, &terminal.Mock{}}
			ctx := terminal.Context(context.Background(), m.term)
			ctx = config.Context(ctx, m.cfg)
			test.init(m)

			FirstRunCheckStats(ctx, test.bannerShown)
			m.cfg.AssertExpectations(t)
			m.term.AssertExpectations(t)
		})
	}
}

func mockShowBanner(m *terminal.Mock) {
	bg := color.New(color.BgMagenta)
	m.On("Writeln", []interface{}(nil)).Return(0, nil).Once()
	m.On("Printf", bg.Sprintf(strings.Repeat(" ", 60)+"\n"), []interface{}(nil)).Return().Once()
	bg.Add(color.FgWhite)
	m.On("Printf",
		bg.Sprintf(strings.Repeat(" ", 16)+"Welcome to Akamai CLI v"+version.Version+strings.Repeat(" ", 16)+"\n"),
		[]interface{}(nil)).Return().Once()
	m.On("Printf", bg.Sprintf(strings.Repeat(" ", 60)+"\n"), []interface{}(nil)).Return()
	m.On("Writeln", []interface{}(nil)).Return(0, nil).Once()
}
