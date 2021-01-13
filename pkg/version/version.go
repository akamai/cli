package version

import "github.com/Masterminds/semver"

const (
	// Version Application Version
	Version = "1.1.5"
)

func Compare(left string, right string) int {
	leftVersion, err := semver.NewVersion(left)
	if err != nil {
		return -2
	}

	rightVersion, err := semver.NewVersion(right)
	if err != nil {
		return 2
	}

	if leftVersion.LessThan(rightVersion) {
		return 1
	} else if leftVersion.GreaterThan(rightVersion) {
		return -1
	}

	return 0
}
