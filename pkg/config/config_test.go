package config

import (
	"context"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/go-ini/ini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestNewIni(t *testing.T) {
	tests := map[string]struct {
		configPath            string
		expectedPath          string
		expectedConfigVersion string
		wantErr               bool
	}{
		"config does not exist in map, create empty config": {
			configPath:            "./testdata/no_config",
			expectedPath:          "testdata/no_config/.akamai-cli/config",
			expectedConfigVersion: "",
		},
		"load config from path": {
			configPath:            "./testdata",
			expectedPath:          "testdata/.akamai-cli/config",
			expectedConfigVersion: "1.1",
		},
		"invalid config": {
			configPath: "./testdata/invalid_config",
			wantErr:    true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", test.configPath))
			defer func() {
				require.NoError(t, os.Unsetenv("AKAMAI_CLI_HOME"))
			}()
			config, err := NewIni()
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expectedPath, config.path)
			section := config.file.Section("cli")
			key := section.Key("config-version")
			assert.Equal(t, test.expectedConfigVersion, key.String())
		})
	}
}

func TestSave(t *testing.T) {
	dir, err := ioutil.TempDir(".", t.Name())
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", dir))
	cfg, err := NewIni()
	require.NoError(t, err)
	ctx := terminal.Context(context.Background(), &terminal.Mock{})
	assert.NoError(t, cfg.Save(ctx))
	_, err = os.Stat(cfg.path)
	assert.NoError(t, err)
}

func TestContext(t *testing.T) {
	cfg := IniConfig{
		path: "test",
		file: ini.Empty(),
	}
	ctx := context.Background()
	ctx = Context(ctx, &cfg)
	got, ok := ctx.Value(configContext).(*IniConfig)
	assert.True(t, ok)
	assert.Equal(t, &cfg, got)
}

func TestGet(t *testing.T) {
	tests := map[string]struct {
		givenConfig Config
		shouldPanic bool
	}{
		"terminal found in context": {
			givenConfig: &IniConfig{path: "test", file: ini.Empty()},
		},
		"terminal not in context": {
			givenConfig: nil,
			shouldPanic: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := Context(context.Background(), test.givenConfig)
			if test.shouldPanic {
				assert.PanicsWithError(t, "context does not have a Config", func() {
					Get(ctx)
				})
				return
			}
			cfg := Get(ctx)
			assert.Equal(t, test.givenConfig, cfg)
		})
	}
}

func TestValues(t *testing.T) {
	tests := map[string]struct {
		givenConfig *IniConfig
		expected    map[string]map[string]string
	}{
		"empty ini file": {
			givenConfig: &IniConfig{path: "test", file: ini.Empty()},
			expected: map[string]map[string]string{
				"DEFAULT": {},
			},
		},
		"load ini file from path": {
			givenConfig: func() *IniConfig {
				require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
				defer func() {
					require.NoError(t, os.Unsetenv("AKAMAI_CLI_HOME"))
				}()
				cfg, err := NewIni()
				require.NoError(t, err)
				return cfg
			}(),
			expected: map[string]map[string]string{
				"DEFAULT": {},
				"cli": {
					"enable-cli-statistics": "",
					"config-version":        "1.1",
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			vals := test.givenConfig.Values()
			assert.Equal(t, test.expected, vals)
		})
	}
}

func TestGetValue(t *testing.T) {
	tests := map[string]struct {
		section, key string
		exists       bool
		value        string
	}{
		"key exists, not empty value": {
			section: "cli",
			key:     "config-version",
			exists:  true,
			value:   "1.1",
		},
		"key exists, empty value": {
			section: "cli",
			key:     "enable-cli-statistics",
			exists:  true,
			value:   "",
		},
		"value does not exist": {
			section: "cli",
			key:     "test",
			exists:  false,
			value:   "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", "./testdata"))
			defer func() {
				require.NoError(t, os.Unsetenv("AKAMAI_CLI_HOME"))
			}()
			cfg, err := NewIni()
			require.NoError(t, err)
			val, ok := cfg.GetValue(test.section, test.key)
			assert.Equal(t, test.exists, ok)
			assert.Equal(t, test.value, val)
		})
	}
}

func TestSetValue(t *testing.T) {
	cfg := IniConfig{path: "test", file: ini.Empty()}
	cfg.SetValue("cli", "testKey", "abc")
	assert.Equal(t, "abc", cfg.file.Section("cli").Key("testKey").String())
}

func TestUnsetValue(t *testing.T) {
	cfg := IniConfig{path: "test", file: ini.Empty()}
	cfg.SetValue("cli", "testKey", "abc")
	cfg.UnsetValue("cli", "testKey")
	_, ok := cfg.GetValue("cli", "testKey")
	assert.False(t, ok)
}

func TestExportConfigEnv(t *testing.T) {
	tests := map[string]struct {
		givenValues      map[string]string
		expectedEnvs     map[string]string
		upgradeFileName  string
		upgradeFileValue string
	}{
		"no migration, current version is latest": {
			givenValues: map[string]string{
				"config-version": "1.1",
				"some-key":       "test",
			},
			expectedEnvs: map[string]string{
				"AKAMAI_CLI_CONFIG_VERSION": "1.1",
				"AKAMAI_CLI_SOME_KEY":       "test",
			},
		},
		"current version is 1, migrate": {
			givenValues: map[string]string{
				"config-version":        "1",
				"enable-cli-statistics": "true",
				"stats-version":         "1.0",
				"some-key":              "test",
			},
			expectedEnvs: map[string]string{
				"AKAMAI_CLI_CONFIG_VERSION":        "1.1",
				"AKAMAI_CLI_SOME_KEY":              "test",
				"AKAMAI_CLI_STATS_VERSION":         "1.0",
				"AKAMAI_CLI_ENABLE_CLI_STATISTICS": "true",
			},
		},
		"no version in config, upgrade to 1.1, no upgrade file": {
			givenValues: map[string]string{
				"enable-cli-statistics": "true",
				"stats-version":         "1.0",
				"some-key":              "test",
			},
			expectedEnvs: map[string]string{
				"AKAMAI_CLI_CONFIG_VERSION":        "1.1",
				"AKAMAI_CLI_SOME_KEY":              "test",
				"AKAMAI_CLI_STATS_VERSION":         "1.0",
				"AKAMAI_CLI_ENABLE_CLI_STATISTICS": "true",
			},
		},
		"no version in config, .upgrade-check file with 'never'": {
			givenValues: map[string]string{
				"enable-cli-statistics": "true",
				"stats-version":         "1.0",
				"some-key":              "test",
			},
			upgradeFileName:  ".upgrade-check",
			upgradeFileValue: "never",
			expectedEnvs: map[string]string{
				"AKAMAI_CLI_CONFIG_VERSION":        "1.1",
				"AKAMAI_CLI_SOME_KEY":              "test",
				"AKAMAI_CLI_STATS_VERSION":         "1.0",
				"AKAMAI_CLI_ENABLE_CLI_STATISTICS": "true",
				"AKAMAI_CLI_LAST_UPGRADE_CHECK":    "never",
			},
		},
		"no version in config, .upgrade-check file with 'ignore''": {
			givenValues: map[string]string{
				"enable-cli-statistics": "true",
				"stats-version":         "1.0",
				"some-key":              "test",
			},
			upgradeFileName:  ".upgrade-check",
			upgradeFileValue: "ignore",
			expectedEnvs: map[string]string{
				"AKAMAI_CLI_CONFIG_VERSION":        "1.1",
				"AKAMAI_CLI_SOME_KEY":              "test",
				"AKAMAI_CLI_STATS_VERSION":         "1.0",
				"AKAMAI_CLI_ENABLE_CLI_STATISTICS": "true",
				"AKAMAI_CLI_LAST_UPGRADE_CHECK":    "ignore",
			},
		},
		"no version in config, .update-check file with 'never'": {
			givenValues: map[string]string{
				"enable-cli-statistics": "true",
				"stats-version":         "1.0",
				"some-key":              "test",
			},
			upgradeFileName:  ".update-check",
			upgradeFileValue: "never",
			expectedEnvs: map[string]string{
				"AKAMAI_CLI_CONFIG_VERSION":        "1.1",
				"AKAMAI_CLI_SOME_KEY":              "test",
				"AKAMAI_CLI_STATS_VERSION":         "1.0",
				"AKAMAI_CLI_ENABLE_CLI_STATISTICS": "true",
				"AKAMAI_CLI_LAST_UPGRADE_CHECK":    "never",
			},
		},
		"no version in config, .upgrade-check file with date": {
			givenValues: map[string]string{
				"enable-cli-statistics": "true",
				"stats-version":         "1.0",
				"some-key":              "test",
			},
			upgradeFileName:  ".upgrade-check",
			upgradeFileValue: "2021-02-03 16:46:43.123456789 +0100 UTC m=123",
			expectedEnvs: map[string]string{
				"AKAMAI_CLI_CONFIG_VERSION":        "1.1",
				"AKAMAI_CLI_SOME_KEY":              "test",
				"AKAMAI_CLI_STATS_VERSION":         "1.0",
				"AKAMAI_CLI_ENABLE_CLI_STATISTICS": "true",
				"AKAMAI_CLI_LAST_UPGRADE_CHECK":    "2021-02-03T16:46:43Z",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dir, err := ioutil.TempDir(".", "test")
			require.NoError(t, err)
			defer func() {
				os.RemoveAll(dir)
			}()
			require.NoError(t, os.Setenv("AKAMAI_CLI_HOME", dir))
			cfg, err := NewIni()
			require.NoError(t, err)
			ctx := terminal.Context(context.Background(), &terminal.Mock{})
			if test.upgradeFileName != "" {
				f, err := os.Create(filepath.Join(dir, ".akamai-cli", test.upgradeFileName))
				require.NoError(t, err)
				_, err = f.WriteString(test.upgradeFileValue)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}
			for k, v := range test.givenValues {
				cfg.SetValue("cli", k, v)
			}
			err = cfg.ExportEnv(ctx)
			require.NoError(t, err)
			for k, v := range test.expectedEnvs {
				assert.Equal(t, v, os.Getenv(k))
				require.NoError(t, os.Unsetenv(k))
			}
		})
	}
}
