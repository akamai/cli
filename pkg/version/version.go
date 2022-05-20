package version

import "github.com/Masterminds/semver"

const (
	// Version Application Version
	Version = "1.5.0"
	// Equals p1==p2 in version.Compare(p1, p2)
	Equals = 0
	// Error failure parsing one of the parameters in version.Compare(p1, p2)
	Error = 2
	// Greater p1>p2 in version.Compare(p1, p2)
	Greater = -1
	// Smaller p1<p2 in version.Compare(p1, p2)
	Smaller = 1
)

// Compare two versions.
//
// * if left < right: return 1 (version.Smaller)
//
// * if left > right: return -1 (version.Greater)
//
// * if left == right: return 0 (version.Equals)
//
// * if unable to parse left or right: return 2 (version.Error)
func Compare(left, right string) int {
	leftVersion, err := semver.NewVersion(left)
	if err != nil {
		return Error
	}

	rightVersion, err := semver.NewVersion(right)
	if err != nil {
		return Error
	}

	if leftVersion.LessThan(rightVersion) {
		return Smaller
	} else if leftVersion.GreaterThan(rightVersion) {
		return Greater
	}

	return Equals
}
