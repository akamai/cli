package commands

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed package_list/package-list.json
var embeddedPackages string

type (
	// packageReader is an interface for reading packages
	packageReader interface {
		readPackage() (*packageList, error)
	}

	// pkgReader contains information about packages
	pkgReader struct {
		content string
	}

	// packageList represents list of packages
	packageList struct {
		Version  float64           `json:"version"`
		Packages []packageListItem `json:"packages"`
	}

	// packageListItem represents contents of a single package
	packageListItem struct {
		Title        string       `json:"title"`
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		URL          string       `json:"url"`
		Issues       string       `json:"issues"`
		Commands     []command    `json:"commands"`
		Requirements requirements `json:"requirements"`
	}

	// requirements represents package requirements
	requirements struct {
		Go     string `json:"go"`
		Php    string `json:"php"`
		Node   string `json:"node"`
		Ruby   string `json:"ruby"`
		Python string `json:"python"`
	}
)

// newPackageReader returns default package reader
func newPackageReader(str string) *pkgReader {
	return &pkgReader{content: str}
}

// readPackage reads packages inside of packages file
func (pr *pkgReader) readPackage() (*packageList, error) {
	packagesList := &packageList{}
	if err := json.Unmarshal([]byte(pr.content), packagesList); err != nil {
		return nil, fmt.Errorf("readPackage: %s", err)
	}

	return packagesList, nil
}
