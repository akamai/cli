package config

import (
	"os"
	"testing"
)

func TestOpenConfig(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name              string
		args              args
		wantConfigVersion string
		wantErr           bool
	}{
		{
			name:              "success - Open valid config path - valid path",
			args:              args{path: "./testdata"},
			wantConfigVersion: "1.1",
			wantErr:           false,
		},
		{
			name:              "fail - Open valid config path - path not provided",
			args:              args{},
			wantConfigVersion: "1.1",
			wantErr:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.path != "" {
				os.Setenv("AKAMAI_CLI_HOME", tt.args.path)
			}

			config, err := OpenConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			section := config.Section("cli")
			key := section.Key("config-version")
			if key.String() != tt.wantConfigVersion {
				t.Errorf("OpenConfig() error = %v, wantConfigVersion %v", err, tt.wantConfigVersion)
				return
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Save valid config path",
			args:    args{path: "./testdata"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("AKAMAI_CLI_HOME", tt.args.path)
			err := SaveConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_migrateConfig(t *testing.T) {
	checkLatestVersion := func() {
		config, err := OpenConfig()
		if err != nil {
			t.Errorf("migrateConfig() %v", err)
			return
		}

		section := config.Section("cli")
		key := section.Key("config-version")
		if key.String() != "1.1" {
			t.Errorf("migrateConfig() got = %v, wantConfigVersion 1.1", key.String())
			return
		}
	}

	t.Run("Migrate config - current version 1.1 vs config version 1.1", func(t *testing.T) {
		migrateConfig()
		checkLatestVersion()
	})

	t.Run("Migrate config - current version 1 vs config version 1.1", func(t *testing.T) {
		os.Setenv("AKAMAI_CLI_HOME", "./testdata")
		SetConfigValue("cli", "config-version", "1")
		migrateConfig()
		checkLatestVersion()
	})

	t.Run("Migrate config - current version empty vs config version 1.1", func(t *testing.T) {
		os.Setenv("AKAMAI_CLI_HOME", "./testdata")
		UnsetConfigValue("cli", "config-version")
		migrateConfig()
		checkLatestVersion()
	})

	t.Run("Migrate config - current version empty, w/upgrade check - never vs config version 1.1", func(t *testing.T) {
		os.Setenv("AKAMAI_CLI_HOME", "./testdata/upgrade")
		UnsetConfigValue("cli", "config-version")

		f, _ := os.Create("./testdata/upgrade/.akamai-cli/.upgrade-check")
		defer f.Close()
		f.Write([]byte("never"))

		migrateConfig()
		checkLatestVersion()
	})

	t.Run("Migrate config - current version empty, w/update check - never vs config version 1.1", func(t *testing.T) {
		os.Setenv("AKAMAI_CLI_HOME", "./testdata/upgrade")
		UnsetConfigValue("cli", "config-version")

		f, _ := os.Create("./testdata/upgrade/.akamai-cli/.update-check")
		defer f.Close()
		f.Write([]byte("never"))

		migrateConfig()
		checkLatestVersion()
	})

	t.Run("Migrate config - current version empty, w/upgrade check vs config version 1.1", func(t *testing.T) {
		os.Setenv("AKAMAI_CLI_HOME", "./testdata/upgrade")
		UnsetConfigValue("cli", "config-version")

		f, _ := os.Create("./testdata/upgrade/.akamai-cli/.upgrade-check")
		defer f.Close()
		f.Write([]byte("2006-01-02 15:04:05.999999999 -0700 MST"))

		migrateConfig()
		checkLatestVersion()
	})

	t.Run("Migrate config - current version empty, w/upgrade check - null vs config version 1.1", func(t *testing.T) {
		os.Setenv("AKAMAI_CLI_HOME", "./testdata/upgrade")
		UnsetConfigValue("cli", "config-version")

		f, _ := os.Create("./testdata/upgrade/.akamai-cli/.upgrade-check")
		defer f.Close()
		f.Write([]byte("nullm="))

		migrateConfig()
		checkLatestVersion()
	})
}

func TestExportConfigEnv(t *testing.T) {
	t.Run("Export config - current version 1 vs config version 1.1", func(t *testing.T) {
		os.Setenv("AKAMAI_CLI_HOME", "./testdata")
		SetConfigValue("cli", "config-version", "1")
		ExportConfigEnv()
	})
}
