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
	"encoding/json"
	"github.com/akamai/cli/pkg/tools"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type CommandPackage struct {
	Commands []command `json:"commands"`

	Requirements struct {
		Go     string `json:"go"`
		Php    string `json:"php"`
		Node   string `json:"node"`
		Ruby   string `json:"ruby"`
		Python string `json:"python"`
	} `json:"requirements"`

	Action interface{} `json:"-"`
}

func ReadPackage(dir string) (CommandPackage, error) {
	if _, err := os.Stat(filepath.Join(dir, "cli.json")); err != nil {
		dir = filepath.Dir(dir)
		if _, err = os.Stat(filepath.Join(dir, "cli.json")); err != nil {
			return CommandPackage{}, cli.NewExitError("Package does not contain a cli.json file.", 1)
		}
	}

	var packageData CommandPackage
	cliJSON, err := ioutil.ReadFile(filepath.Join(dir, "cli.json"))
	if err != nil {
		return CommandPackage{}, err
	}

	err = json.Unmarshal(cliJSON, &packageData)
	if err != nil {
		return CommandPackage{}, err
	}

	for key := range packageData.Commands {
		packageData.Commands[key].Name = strings.ToLower(packageData.Commands[key].Name)
	}

	return packageData, nil
}

func GetPackagePaths() []string {
	akamaiCliPath, err := tools.GetAkamaiCliSrcPath()
	if err == nil && akamaiCliPath != "" {
		paths, _ := filepath.Glob(filepath.Join(akamaiCliPath, "*"))
		if len(paths) > 0 {
			return paths
		}
	}

	return []string{}
}

func FindPackageDir(dir string) string {
	if stat, err := os.Stat(dir); err == nil && stat != nil && !stat.IsDir() {
		dir = filepath.Dir(dir)
	}

	if _, err := os.Stat(filepath.Join(dir, "cli.json")); err != nil {
		if os.IsNotExist(err) {
			if filepath.Dir(dir) == "" {
				return ""
			}

			return FindPackageDir(filepath.Dir(dir))
		}
	}

	return dir
}

func DetermineCommandLanguage(cmdPackage CommandPackage) string {
	if cmdPackage.Requirements.Php != "" {
		return "php"
	}

	if cmdPackage.Requirements.Node != "" {
		return "javascript"
	}

	if cmdPackage.Requirements.Ruby != "" {
		return "ruby"
	}

	if cmdPackage.Requirements.Go != "" {
		return "go"
	}

	if cmdPackage.Requirements.Python != "" {
		return "python"
	}

	return ""
}

func DownloadBin(dir string, cmd command) bool {
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
		log.Tracef("Unable to create URL. Template: %s; Error: %s.", cmd.Bin, err.Error())
		return false
	}

	url := buf.String()
	log.Tracef("Fetching binary from %s", url)

	bin, err := os.Create(filepath.Join(dir, "akamai-"+strings.ToLower(cmd.Name)+cmd.BinSuffix))
	bin.Chmod(0775)
	if err != nil {
		return false
	}
	defer bin.Close()

	res, err := http.Get(url)
	if err != nil {
		return false
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return false
	}

	n, err := io.Copy(bin, res.Body)
	if err != nil || n == 0 {
		return false
	}

	return true
}
