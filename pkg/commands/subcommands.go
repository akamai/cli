// Copyright 2018. Akamai Technologies, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/akamai/cli/pkg/log"
	"github.com/akamai/cli/pkg/packages"
	"github.com/akamai/cli/pkg/tools"
	"github.com/urfave/cli/v2"
)

type subcommands struct {
	Commands     []command                     `json:"commands"`
	Requirements packages.LanguageRequirements `json:"requirements"`
	Action       cli.ActionFunc                `json:"-"`
	Pkg          string                        `json:"pkg"`
}

func readPackage(dir string) (subcommands, error) {
	if _, err := os.Stat(filepath.Join(dir, "cli.json")); err != nil {
		dir = filepath.Dir(dir)
		if _, err = os.Stat(filepath.Join(dir, "cli.json")); err != nil {
			return subcommands{}, cli.Exit("Package does not contain a cli.json file.", 1)
		}
	}

	var packageData subcommands
	cliJSON, err := ioutil.ReadFile(filepath.Join(dir, "cli.json"))
	if err != nil {
		return subcommands{}, err
	}

	err = json.Unmarshal(cliJSON, &packageData)
	if err != nil {
		return subcommands{}, err
	}

	for key := range packageData.Commands {
		packageData.Commands[key].Name = strings.ToLower(packageData.Commands[key].Name)
	}

	packageData.Pkg = filepath.Base(strings.Replace(dir, "cli-", "", 1))

	return packageData, nil
}

func getPackagePaths() []string {
	akamaiCliPath, err := tools.GetAkamaiCliSrcPath()
	if err == nil && akamaiCliPath != "" {
		paths, _ := filepath.Glob(filepath.Join(akamaiCliPath, "*"))
		if len(paths) > 0 {
			return paths
		}
	}

	return []string{}
}

func findPackageDir(dir string) string {
	if stat, err := os.Stat(dir); err == nil && stat != nil && !stat.IsDir() {
		dir = filepath.Dir(dir)
	}

	if _, err := os.Stat(filepath.Join(dir, "cli.json")); err != nil {
		if os.IsNotExist(err) {
			if filepath.Dir(dir) == "" || filepath.Dir(dir) == "." {
				return ""
			}

			return findPackageDir(filepath.Dir(dir))
		}
	}

	return dir
}

func downloadBin(ctx context.Context, dir string, cmd command) error {
	logger := log.FromContext(ctx)
	cmd.Arch = runtime.GOARCH

	cmd.OS = runtime.GOOS
	if runtime.GOOS == "darwin" {
		cmd.OS = "mac"
	}

	if runtime.GOOS == "windows" {
		cmd.BinSuffix = ".exe"
	}

	t := template.Must(template.New("url").Parse(cmd.Bin))
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, cmd); err != nil {
		logger.Debugf("Unable to create URL. Template: %s; Error: %s.", cmd.Bin, err.Error())
		return err
	}

	url := buf.String()
	logger.Debugf("Fetching binary from %s", url)

	binName := filepath.Join(dir, "akamai-"+strings.ToLower(cmd.Name)+cmd.BinSuffix)
	bin, err := os.Create(binName)
	if err != nil {
		return err
	}
	defer func() {
		if err := bin.Close(); err != nil {
			logger.Errorf("Error closing file: %s", err)
		}
	}()

	if err := os.Chmod(binName, 0775); err != nil {
		return err
	}

	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			logger.Errorf("Error closing request body: %s", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid response status while fetching command binary: %d", res.StatusCode)
	}

	n, err := io.Copy(bin, res.Body)
	if err != nil || n == 0 {
		return err
	}

	return nil
}
