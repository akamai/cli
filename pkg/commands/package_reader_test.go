package commands

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageReader(t *testing.T) {
	bytes, err := ioutil.ReadFile("testdata/test_packages/sample_packages.json")
	require.NoError(t, err)

	pr := newPackageReader(string(bytes))
	packages, err := pr.readPackage()
	require.NoError(t, err)

	expectedPackages := &packageList{
		Version: 1.0,
		Packages: []packageListItem{
			{
				Title: "Test title 1",
				Name:  "Test name 1",
				Commands: []command{
					{
						Name:        "Test 1",
						Version:     "1.1.0",
						Description: "Test description 1",
					},
				},
				Requirements: requirements{
					Node: "7.0.0",
				},
			},
			{
				Title:   "Test title 2",
				Name:    "Test name 2",
				Version: "2.0.0",
				URL:     "test.url.com",
				Issues:  "issue 1",
				Commands: []command{
					{
						Name:        "test 2",
						Aliases:     []string{"test", "test2"},
						Version:     "2.5.0",
						Description: "Test description 2",
					},
				},
				Requirements: requirements{
					Go: "1.17.0",
				},
			},
		},
	}

	assert.Equal(t, expectedPackages, packages)
}

func TestActualPackageReader(t *testing.T) {
	pr := newPackageReader(embeddedPackages)
	_, err := pr.readPackage()
	require.NoError(t, err)
}

func TestUnmarshalPackage(t *testing.T) {
	file, err := os.ReadFile("testdata/test_packages/sample_packages.json")
	require.NoError(t, err)

	result := &packageList{}
	err = json.Unmarshal(file, result)
	require.NoError(t, err)

	expectedPackageList := &packageList{
		Version: 1.0,
		Packages: []packageListItem{
			{
				Title:   "Test title 1",
				Name:    "Test name 1",
				Version: "",
				URL:     "",
				Issues:  "",
				Commands: []command{
					{
						Name:        "Test 1",
						Aliases:     nil,
						Version:     "1.1.0",
						Description: "Test description 1",
					},
				},
				Requirements: struct {
					Go     string `json:"go"`
					Php    string `json:"php"`
					Node   string `json:"node"`
					Ruby   string `json:"ruby"`
					Python string `json:"python"`
				}{
					Node: "7.0.0",
				},
			},
			{
				Title:   "Test title 2",
				Name:    "Test name 2",
				Version: "2.0.0",
				URL:     "test.url.com",
				Issues:  "issue 1",
				Commands: []command{
					{
						Name: "test 2",
						Aliases: []string{
							"test", "test2",
						},
						Version:      "2.5.0",
						Description:  "Test description 2",
						Usage:        "",
						Arguments:    "",
						Bin:          "",
						AutoComplete: false,
						LdFlags:      "",
						Flags:        nil,
						Docs:         "",
						BinSuffix:    "",
						OS:           "",
						Arch:         "",
						Subcommands:  nil,
					},
				},
				Requirements: struct {
					Go     string `json:"go"`
					Php    string `json:"php"`
					Node   string `json:"node"`
					Ruby   string `json:"ruby"`
					Python string `json:"python"`
				}{
					Go: "1.17.0",
				},
			},
		},
	}

	assert.Equal(t, expectedPackageList, result)
}
