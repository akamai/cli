//+build noautoupgrade

package main

func checkForUpgrade(force bool) string {
	return ""
}

func getLatestReleaseVersion() string {
	return "0"
}

func upgradeCli(latestVersion string) bool {
	return false
}

func getUpgradeCommand() *commandPackage {
	return nil
}
