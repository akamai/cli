package terminal

import (
	"fmt"
	spnr "github.com/briandowns/spinner"
	"io"
	"strings"
)

type (
	// Spinner contains methods to operate on spinner
	Spinner interface {
		io.Writer
		Start(f string, args ...interface{})
		Stop(status SpinnerStatus)
		OK()
		WarnOK()
		Warn()
		Fail()
	}

	// DefaultSpinner defines a simple status spinner
	DefaultSpinner struct {
		spinner *spnr.Spinner
		prefix  string
	}
)

// Start starts the spinner using the provided string as the prefix
func (s *DefaultSpinner) Start(f string, args ...interface{}) {
	s.prefix = fmt.Sprintf(f, args...)
	s.spinner.Prefix = s.prefix + " "
	s.spinner.Start()
}

// Stop stops the spinner and updates the final status message
func (s *DefaultSpinner) Stop(status SpinnerStatus) {
	s.spinner.Suffix = ""
	s.spinner.FinalMSG = s.prefix + " " + string(status)
	s.spinner.Stop()
}

// Write implements the io.Writer interface and updates the suffix of the spinner
func (s *DefaultSpinner) Write(v []byte) (n int, err error) {
	s.spinner.Suffix = " " + strings.TrimSpace(string(v))
	return len(v), nil
}

// OK stops the spinner with ok status
func (s *DefaultSpinner) OK() {
	s.Stop(SpinnerStatusOK)
}

// WarnOK stops the spinner with WarnOK status
func (s *DefaultSpinner) WarnOK() {
	s.Stop(SpinnerStatusWarnOK)
}

// Warn stops the spinner with Warn status
func (s *DefaultSpinner) Warn() {
	s.Stop(SpinnerStatusWarn)
}

// Fail stops the spinner with fail status
func (s *DefaultSpinner) Fail() {
	s.Stop(SpinnerStatusFail)
}
