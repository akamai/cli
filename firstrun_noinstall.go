//+build nofirstrun

package main

func firstRun() error {
	checkStats(false)
	return nil
}
