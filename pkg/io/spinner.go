package io

import (
	"fmt"
	"github.com/akamai/cli/pkg/app"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"os"
	"time"
)

func StartSpinner(prefix string, finalMsg string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[26], 500*time.Millisecond)
	s.Writer = app.App.ErrWriter
	s.Prefix = prefix
	s.FinalMSG = finalMsg
	if log := os.Getenv("AKAMAI_LOG"); len(log) > 0 || !isTTY() {
		fmt.Println(prefix)
	} else {
		s.Start()
	}

	return s
}

func StopSpinner(s *spinner.Spinner, finalMsg string, usePrefix bool) {
	if s == nil {
		return
	}
	if usePrefix {
		s.FinalMSG = s.Prefix + finalMsg
	} else {
		s.FinalMSG = finalMsg
	}

	if log := os.Getenv("AKAMAI_LOG"); len(log) > 0 || !isTTY() {
		fmt.Println(s.FinalMSG)
		return
	}
	s.Stop()
}

func StopSpinnerOk(s *spinner.Spinner) {
	StopSpinner(s, fmt.Sprintf("... [%s]\n", color.GreenString("OK")), true)
}

func StopSpinnerWarnOk(s *spinner.Spinner) {
	StopSpinner(s, fmt.Sprintf("... [%s]\n", color.CyanString("OK")), true)
}

func StopSpinnerWarn(s *spinner.Spinner) {
	StopSpinner(s, fmt.Sprintf("... [%s]\n", color.CyanString("WARN")), true)
}

func StopSpinnerFail(s *spinner.Spinner) {
	StopSpinner(s, fmt.Sprintf("... [%s]\n", color.RedString("FAIL")), true)
}

func isTTY() bool {
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return false
	}

	return true
}
