// Copyright 2020. Akamai Technologies, Inc
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

package terminal

import (
	"fmt"
	"io"
	"strings"
	"time"

	"os"

	spnr "github.com/briandowns/spinner"
	"github.com/fatih/color"

	"github.com/AlecAivazis/survey/v2"
)

type (
	// Terminal defines a terminal abstration interface
	Terminal interface {
		io.Writer

		// Writef writes a formatted message to the output stream
		Writef(f string, args ...interface{})

		// WriteError write a message to the error stream
		WriteError(interface{})

		// WriteErrorf writes a formatted message to the error stream
		WriteErrorf(f string, args ...interface{})

		// Prompt prompts the use for an open or multiple choice anwswer
		Prompt(p string, options ...string) (string, error)

		// Confirm asks the user for a Y/n response, with a default
		Confirm(p string, d bool) (bool, error)

		// Spinner creates a spinner using the output stream
		Spinner() *Spinner
	}

	// Writer provides a minimal interface for Stdin.
	Writer interface {
		io.Writer
		Fd() uintptr
	}

	// Reader provides a minimal interface for Stdout.
	Reader interface {
		io.Reader
		Fd() uintptr
	}

	terminal struct {
		Out   Writer
		Err   io.Writer
		In    Reader
		start time.Time
	}

	// SpinnerStatus defines a spinner status message
	SpinnerStatus string

	// Spinner defines a simple status spinner
	Spinner struct {
		*spnr.Spinner
		prefix string
	}
)

// SpinnerStatus strings
var (
	SpinnerStatusOK     = SpinnerStatus(fmt.Sprintf("... [%s]\n", color.GreenString("OK")))
	SpinnerStatusWarnOK = SpinnerStatus(fmt.Sprintf("... [%s]\n", color.CyanString("OK")))
	SpinnerStatusWarn   = SpinnerStatus(fmt.Sprintf("... [%s]\n", color.CyanString("WARN")))
	SpinnerStatusFail   = SpinnerStatus(fmt.Sprintf("... [%s]\n", color.RedString("FAIL")))
)

// Standard returns the standard terminal
func Standard() Terminal {
	return terminal{
		Out:   os.Stdout,
		Err:   os.Stderr,
		In:    os.Stdin,
		start: time.Now(),
	}
}

// New returns a new terminal with the specifed streams
func New(out Writer, in Reader, err io.Writer) Terminal {
	return terminal{
		Out:   out,
		Err:   err,
		In:    in,
		start: time.Now(),
	}
}

func (t terminal) Writef(f string, args ...interface{}) {
	t.Write([]byte(fmt.Sprintf(f, args...)))
	fmt.Fprintln(t.Out)
}

func (t terminal) Write(v []byte) (n int, err error) {
	msg := string(v)
	return t.Out.Write([]byte(msg))
}

func (t terminal) WriteErrorf(f string, args ...interface{}) {
	t.Err.Write([]byte(fmt.Sprintf(f, args...)))
}

func (t terminal) WriteError(v interface{}) {
	t.Err.Write([]byte(fmt.Sprint(v)))
}

func (t terminal) Prompt(p string, options ...string) (string, error) {
	q := survey.Question{
		Name:     "q",
		Prompt:   &survey.Input{Message: p},
		Validate: survey.Required,
	}

	if len(options) > 0 {
		q.Prompt = &survey.Select{
			Message: p,
			Options: options,
		}
	}

	answers := struct {
		Q string
	}{}

	err := survey.Ask([]*survey.Question{&q}, &answers, survey.WithStdio(t.In, t.Out, t.Err))
	if err != nil {
		return "", err
	}

	return answers.Q, nil
}

func (t terminal) Confirm(p string, def bool) (bool, error) {
	rval := def

	q := &survey.Confirm{
		Message: p,
		Default: def,
	}

	err := survey.AskOne(q, &rval, survey.WithStdio(t.In, t.Out, t.Err))

	return rval, err
}

func (t terminal) Spinner() *Spinner {
	s := spnr.New(spnr.CharSets[33], 500*time.Millisecond)
	s.Writer = t

	return &Spinner{
		Spinner: s,
	}
}

// Start starts the spinner using the provided string as the prefix
func (s *Spinner) Start(f string, args ...interface{}) {
	s.prefix = fmt.Sprintf(f, args...)
	s.Spinner.Prefix = s.prefix + " "
	s.Spinner.Start()
}

// Stop stops the spinner and updates the final status message
func (s *Spinner) Stop(status SpinnerStatus) {
	s.Spinner.Suffix = ""
	s.Spinner.FinalMSG = s.prefix + " " + string(status)
	s.Spinner.Stop()
}

// Write implements the io.Writer interface and updates the suffix of the spinner
func (s *Spinner) Write(v []byte) (n int, err error) {
	s.Spinner.Suffix = " " + strings.TrimSpace(string(v))
	return len(v), nil
}

// DiscardWriter returns a discard write that direct output to /dev/null
func DiscardWriter() Writer {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}
	return f
}
