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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/akamai/cli/v2/pkg/color"
	"github.com/akamai/cli/v2/pkg/log"
	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/akamai/cli/v2/pkg/tools"
	"github.com/urfave/cli/v2"
)

var (
	githubURLTemplate = "https://raw.githubusercontent.com/akamai/%s/master/cli.json"
)

func cmdSearch(c *cli.Context) (e error) {
	pr := newPackageReader(embeddedPackages)
	return cmdSearchWithPackageReader(c, pr)
}

func cmdSearchWithPackageReader(c *cli.Context, pr packageReader) (e error) {
	c.Context = log.WithCommandContext(c.Context, c.Command.Name)
	start := time.Now()
	logger := log.FromContext(c.Context)
	logger.Debug("SEARCH START")
	defer func() {
		if e == nil {
			logger.Debug(fmt.Sprintf("SEARCH FINISHED: %v", time.Since(start)))
		} else {
			logger.Error(fmt.Sprintf("SEARCH ERROR: %v", e.Error()))
		}
	}()
	if !c.Args().Present() {
		return cli.Exit(color.RedString("You must specify one or more keywords"), 1)
	}

	packages, err := pr.readPackage()
	if err != nil {
		return cli.Exit(color.RedString(err.Error()), 1)
	}

	err = searchPackages(c.Context, c.Args().Slice(), packages)
	if err != nil {
		return cli.Exit(color.RedString(err.Error()), 1)
	}

	return nil
}

func searchPackages(ctx context.Context, keywords []string, packageList *packageList) error {
	results := make(map[int]map[string]packageListItem)

	term := terminal.Get(ctx)

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
				results[hits] = make(map[string]packageListItem)
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

	term.Printf(color.YellowString("Results Found:")+" %d\n\n", len(resultPkgs))

	return printResult(resultHits, resultPkgs, results, term)
}

func printResult(resultHits []int, resultPkgs []string, results map[int]map[string]packageListItem, term terminal.Terminal) error {
	var installedVersion, availableVersion string
	for _, hits := range resultHits {
		for _, pkgName := range resultPkgs {
			if _, ok := results[hits][pkgName]; ok {
				pkg := results[hits][pkgName]
				term.Printf(color.GreenString("Package: ")+"%s [%s]\n", pkg.Title, color.BlueString(pkg.Name))
				for _, cmd := range pkg.Commands {
					var aliases string
					if len(cmd.Aliases) == 1 {
						aliases = fmt.Sprintf("(alias: %s)", cmd.Aliases[0])
					} else if len(cmd.Aliases) > 1 {
						aliases = fmt.Sprintf("(aliases: %s)", strings.Join(cmd.Aliases, ", "))
					}
					term.Printf(color.BoldString("  Command:")+" %s %s\n", cmd.Name, aliases)

					url := pkg.URL
					var err error
					availableVersion, err = getLatestVersion(url)
					if err != nil {
						return cli.Exit(color.RedString(err.Error()), 1)
					}
					term.Printf(color.BoldString("  Available Version:")+" %s\n", availableVersion)
					installedVersion, err = getVersionFromSystem(pkg.Name)
					if err != nil {
						return cli.Exit(color.RedString(err.Error()), 1)
					}
					if installedVersion != "" {
						term.Printf(color.BoldString("  Installed Version:")+" %s\n", installedVersion)
					}
					term.Printf(color.BoldString("  Description:")+" %s\n\n", cmd.Description)
				}
			}
		}
	}

	if len(resultHits) > 0 {
		if installedVersion == "" {
			term.Printf("\nInstall using \"%s\".\n", color.BlueString("%s install [package]", tools.Self()))
		} else if installedVersion != availableVersion {
			term.Printf("\nUpdate using \"%s\".\n", color.BlueString("%s update [package]", tools.Self()))
		} else {
			term.Printf(color.BlueString("Package is already up-to-date on your system"))
		}
	}
	return nil
}

func getLatestVersion(s string) (string, error) {

	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("error parsing URL: %s", err.Error())
	}

	// extract the last string of the package URL
	lastSegment := path.Base(u.Path)

	repoURL := fmt.Sprintf(githubURLTemplate, lastSegment)
	resp, err := http.Get(repoURL)
	if err != nil {
		return "", fmt.Errorf("error fetching the URL: %s", err.Error())
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Println("error closing the response body:", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading the response body: %w", err)
	}

	var cli CLI
	if err := json.Unmarshal(body, &cli); err != nil {
		return "", fmt.Errorf("error parsing the JSON: %w", err)
	}

	if len(cli.CommandList) > 0 {
		return cli.CommandList[0].Version, nil
	}
	return "", fmt.Errorf("no latest version found")
}

// CLI struct represents an individual command object in package-list.json
type CLI struct {
	CommandList []CommandObject `json:"commands"`
}

// CommandObject contains details for particular command
type CommandObject struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

func getVersionFromSystem(command string) (string, error) {
	paths := filepath.SplitList(getPackageBinPaths())
	suffix := "cli-" + command
	finalPath := ""
	for _, path := range paths {
		if strings.HasSuffix(path, suffix) {
			finalPath = path
			break
		}
	}

	if finalPath == "" {
		return "", nil
	}
	body, err := os.ReadFile(filepath.Join(finalPath, "cli.json"))
	if err != nil {
		return "", fmt.Errorf("Error reading the file: %s", err.Error())

	}

	var cli CLI
	if err := json.Unmarshal(body, &cli); err != nil {
		return "", fmt.Errorf("Error parsing the JSON: %s", err.Error())
	}

	return cli.CommandList[0].Version, nil
}
