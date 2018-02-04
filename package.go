package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/urfave/cli"
)

type commandPackage struct {
	Commands []Command `json:"commands"`

	Requirements struct {
		Go     string `json:"go"`
		Php    string `json:"php"`
		Node   string `json:"node"`
		Ruby   string `json:"ruby"`
		Python string `json:"python"`
	} `json:"requirements"`

	action interface{}
}

func readPackage(dir string) (commandPackage, error) {
	if _, err := os.Stat(filepath.Join(dir, "cli.json")); err != nil {
		dir = path.Dir(dir)
		if _, err = os.Stat(filepath.Join(dir, "cli.json")); err != nil {
			return commandPackage{}, cli.NewExitError("Package does not contain a cli.json file.", 1)
		}
	}

	var packageData commandPackage
	cliJson, err := ioutil.ReadFile(filepath.Join(dir, "cli.json"))
	if err != nil {
		return commandPackage{}, err
	}

	err = json.Unmarshal(cliJson, &packageData)
	if err != nil {
		return commandPackage{}, err
	}

	for key := range packageData.Commands {
		packageData.Commands[key].Name = strings.ToLower(packageData.Commands[key].Name)
	}

	return packageData, nil
}

func getPackagePaths() string {
	path := ""
	akamaiCliPath, err := getAkamaiCliSrcPath()
	if err == nil && akamaiCliPath != "" {
		paths, _ := filepath.Glob(filepath.Join(akamaiCliPath, "*"))
		if len(paths) > 0 {
			path += strings.Join(paths, string(os.PathListSeparator))
		}
	}

	return path
}

func getPackageBinPaths() string {
	path := ""
	akamaiCliPath, err := getAkamaiCliSrcPath()
	if err == nil && akamaiCliPath != "" {
		paths, _ := filepath.Glob(filepath.Join(akamaiCliPath, "*"))
		if len(paths) > 0 {
			path += strings.Join(paths, string(os.PathListSeparator))
		}
		paths, _ = filepath.Glob(filepath.Join(akamaiCliPath, "*", "bin"))
		if len(paths) > 0 {
			path += string(os.PathListSeparator) + strings.Join(paths, string(os.PathListSeparator))
		}
	}

	return path
}


func findPackageDir(dir string) string {
	if stat, err :=  os.Stat(dir); err == nil && stat != nil && !stat.IsDir() {
		dir = path.Dir(dir)
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		if os.IsNotExist(err) {
			if path.Dir(dir) == "" {
				return ""
			}

			return findPackageDir(filepath.Dir(dir))
		}
	}

	return dir
}


func determineCommandLanguage(cmdPackage commandPackage) string {
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

func downloadBin(dir string, cmd Command) bool {
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
		return false
	}

	url := buf.String()

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