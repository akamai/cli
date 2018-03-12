package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
	Commands     []Command `json:"commands"`
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
	repo := "https://developer.akamai.com/cli/package-list"
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
		for _, keyword := range keywords {
			keyword = strings.ToLower(keyword)
			if strings.Contains(strings.ToLower(pkg.Name), keyword) {
				hits += 100
			}

			if strings.Contains(strings.ToLower(pkg.Title), keyword) {
				hits += 50
			}

			valid_cmds := make([]Command, 0)
			for _, cmd := range pkg.Commands {
				cmdMatches := false
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
					hits += 1
					cmdMatches = true
				}

				if cmdMatches {
					valid_cmds = append(valid_cmds, cmd)
				}
			}

			packageList.Packages[key].Commands = valid_cmds
		}

		if hits > 0 {
			if _, ok := results[hits]; !ok {
				results[hits] = make(map[string]packageListPackage)
			}
			results[hits][pkg.Name] = pkg
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

	color.Yellow("Results Found: %d\n\n", len(resultPkgs))

	for _, hits := range resultHits {
		for _, pkg_name := range resultPkgs {
			if _, ok := results[hits][pkg_name]; ok {
				pkg := results[hits][pkg_name]
				color.Green("Package: %s (%s) (rank: %d)\n", pkg.Title, pkg.Name, hits)
				for _, cmd := range results[hits][pkg_name].Commands {
					var aliases string
					if len(cmd.Aliases) == 1 {
						aliases = fmt.Sprintf("(alias: %s)", cmd.Aliases[0])
					} else if len(cmd.Aliases) > 1 {
						aliases = fmt.Sprintf("(aliases: %s)", strings.Join(cmd.Aliases, ", "))
					}

					bold.Printf("    Command: %s %s\n", cmd.Name, aliases)
					fmt.Printf("        %s\n\n", cmd.Description)
				}
			}
		}
	}

	return nil
}
