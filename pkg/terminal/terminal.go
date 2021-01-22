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
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"os"

	spnr "github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"

	"github.com/AlecAivazis/survey/v2"
)

type (
	// Terminal defines a terminal abstration interface
	Terminal interface {
		io.Writer

		// Printf writes a formatted message to the output stream
		Printf(f string, args ...interface{})

		// Writeln writes a line to the terminal
		Writeln(args ...interface{}) (int, error)

		// WriteError write a message to the error stream
		WriteError(interface{})

		// WriteErrorf writes a formatted message to the error stream
		WriteErrorf(f string, args ...interface{})

		// Prompt prompts the use for an open or multiple choice anwswer
		Prompt(p string, options ...string) (string, error)

		// Confirm asks the user for a Y/n response, with a default
		Confirm(p string, d bool) (bool, error)

		// Spinner returns the terminal spinner
		Spinner() *Spinner

		// Error return the error writer
		Error() io.Writer
	}

	// Writer provides a minimal interface for Stdin.
	Writer interface {
		io.Writer
		Fd() uintptr
	}

	colorWriter struct {
		io.Writer
		fd uintptr
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
		spnr  *Spinner
	}

	// SpinnerStatus defines a spinner status message
	SpinnerStatus string

	// Spinner defines a simple status spinner
	Spinner struct {
		*spnr.Spinner
		prefix string
	}

	contextType string
)

var (
	SpinnerStatusOK     = SpinnerStatus(fmt.Sprintf("... [%s]\n", color.GreenString("OK")))
	SpinnerStatusWarnOK = SpinnerStatus(fmt.Sprintf("... [%s]\n", color.CyanString("OK")))
	SpinnerStatusWarn   = SpinnerStatus(fmt.Sprintf("... [%s]\n", color.CyanString("WARN")))
	SpinnerStatusFail   = SpinnerStatus(fmt.Sprintf("... [%s]\n", color.RedString("FAIL")))

	terminalContext contextType = "terminal"
)

// Standard returns the standard terminal
func Standard() Terminal {
	return New(os.Stdout, os.Stdin, os.Stderr)
}

// Color returns a colorable terminal
func Color() Terminal {
	wr := &colorWriter{
		Writer: colorable.NewColorableStdout(),
		fd:     os.Stdout.Fd(),
	}

	return New(wr, os.Stdin, colorable.NewColorableStderr())
}

// New returns a new terminal with the specifed streams
func New(out Writer, in Reader, err io.Writer) Terminal {
	t := terminal{
		Out:   out,
		Err:   err,
		In:    in,
		start: time.Now(),
		spnr:  &Spinner{Spinner: spnr.New(spnr.CharSets[33], 500*time.Millisecond)},
	}

	t.spnr.Writer = t

	return &t
}

func (t terminal) Printf(f string, args ...interface{}) {
	t.Write([]byte(fmt.Sprintf(f, args...)))
}

func (t terminal) Writeln(args ...interface{}) (int, error) {
	return fmt.Fprintln(t, args...)
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

func (t terminal) Error() io.Writer {
	return t.Err
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

func (t *terminal) Spinner() *Spinner {
	return t.spnr
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

// Context sets the terminal in the context
func Context(ctx context.Context, term ...Terminal) context.Context {
	if len(term) > 0 {
		return context.WithValue(ctx, terminalContext, term[0])
	}
	return context.WithValue(ctx, terminalContext, Color())
}

// Get gets the terminal from the context
func Get(ctx context.Context) Terminal {
	t, ok := ctx.Value(terminalContext).(Terminal)
	if !ok {
		panic(errors.New("context does not have a terminal"))
	}

	return t
}

func (w *colorWriter) Fd() uintptr {
	return w.fd
}
