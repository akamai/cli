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
	"encoding/json"
	"fmt"
	"github.com/akamai/cli/pkg/app"
	"github.com/akamai/cli/pkg/tools"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

type packageList struct {
	Version  float64              `json:"version"`
	Packages []packageListPackage `json:"packages"`
}

type packageListPackage struct {
	Title        string    `json:"title"`
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	URL          string    `json:"url"`
	Issues       string    `json:"issues"`
	Commands     []command `json:"commands"`
	Requirements struct {
		Go     string `json:"go"`
		Php    string `json:"php"`
		Node   string `json:"node"`
		Ruby   string `json:"ruby"`
		Python string `json:"python"`
	} `json:"requirements"`
}

func cmdSearch(c *cli.Context) error {
	if !c.Args().Present() {
		return cli.NewExitError(color.RedString("You must specify one or more keywords"), 1)
	}

	packageList, err := fetchPackageList()
	if err != nil {
		return cli.NewExitError(color.RedString(err.Error()), 1)
	}

	err = searchPackages(c.Args(), packageList)
	if err != nil {
		return cli.NewExitError(color.RedString(err.Error()), 1)
	}

	return nil
}

func fetchPackageList() (*packageList, error) {
	var repo string
	if repo = os.Getenv("AKAMAI_CLI_PACKAGE_REPO"); repo == "" {
		repo = "https://developer.akamai.com/cli/package-list.json"
	}
	resp, err := http.Get(repo)
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch remote Package List (%s)", err.Error())
	}

	defer resp.Body.Close()

	result := &packageList{}
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, result)
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch remote Package List (%s)", err.Error())
	}

	return result, nil
}

func searchPackages(keywords []string, packageList *packageList) error {
	results := make(map[int]map[string]packageListPackage)

	var hits int
	for key, pkg := range packageList.Packages {
		hits = 0
		validCmds := make([]command, 0)
		for _, keyword := range keywords {
			keyword = strings.ToLower(keyword)
			if strings.Contains(strings.ToLower(pkg.Name), keyword) {
				hits += 100
			}

			if strings.Contains(strings.ToLower(pkg.Title), keyword) {
				hits += 50
			}
		}

		for _, cmd := range pkg.Commands {
			cmdMatches := false
			for _, keyword := range keywords {
				keyword = strings.ToLower(keyword)

				if strings.Contains(strings.ToLower(cmd.Name), keyword) {
					hits += 30
					cmdMatches = true
				}

				for _, alias := range cmd.Aliases {
					if strings.Contains(strings.ToLower(alias), keyword) {
						hits += 20
						cmdMatches = true
					}
				}

				if strings.Contains(strings.ToLower(cmd.Description), keyword) {
					hits++
					cmdMatches = true
				}
			}

			if cmdMatches {
				validCmds = append(validCmds, cmd)
			}
		}
		packageList.Packages[key].Commands = validCmds

		if hits > 0 {
			if _, ok := results[hits]; !ok {
				results[hits] = make(map[string]packageListPackage)
			}
			results[hits][pkg.Name] = packageList.Packages[key]
		}
	}

	resultHits := make([]int, 0)
	resultPkgs := make([]string, 0)
	for hits := range results {
		resultHits = append(resultHits, hits)
		for _, pkg := range results[hits] {
			resultPkgs = append(resultPkgs, pkg.Name)
		}
	}

	sort.Sort(sort.Reverse(sort.IntSlice(resultHits)))
	sort.Strings(resultPkgs)
	bold := color.New(color.FgWhite, color.Bold)

	fmt.Fprintf(app.App.Writer, color.YellowString("Results Found:")+" %d\n\n", len(resultPkgs))

	for _, hits := range resultHits {
		for _, pkgName := range resultPkgs {
			if _, ok := results[hits][pkgName]; ok {
				pkg := results[hits][pkgName]
				fmt.Fprintf(app.App.Writer, color.GreenString("Package: ")+"%s [%s]\n", pkg.Title, color.BlueString(pkg.Name))
				for _, cmd := range results[hits][pkgName].Commands {
					var aliases string
					if len(cmd.Aliases) == 1 {
						aliases = fmt.Sprintf("(alias: %s)", cmd.Aliases[0])
					} else if len(cmd.Aliases) > 1 {
						aliases = fmt.Sprintf("(aliases: %s)", strings.Join(cmd.Aliases, ", "))
					}

					fmt.Fprintf(app.App.Writer, bold.Sprintf("  Command:")+" %s %s\n", cmd.Name, aliases)
					fmt.Fprintf(app.App.Writer, bold.Sprintf("  Version:")+" %s\n", cmd.Version)
					fmt.Fprintf(app.App.Writer, bold.Sprintf("  Description:")+" %s\n\n", cmd.Description)
				}
			}
		}
	}

	if len(resultHits) > 0 {
		fmt.Fprintf(app.App.Writer, "\nInstall using \"%s\".\n", color.BlueString("%s install [package]", tools.Self()))
	}

	return nil
}
