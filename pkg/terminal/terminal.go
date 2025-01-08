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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/akamai/cli/pkg/color"
	"github.com/akamai/cli/pkg/version"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

type (
	// Terminal defines a terminal abstraction interface
	Terminal interface {
		TermWriter
		Prompter
		Spinner() Spinner
		Error() io.Writer
		IsTTY() bool
	}

	// TermWriter contains methods for basic terminal write operations
	TermWriter interface {
		io.Writer
		Printf(f string, args ...interface{})
		Writeln(args ...interface{}) (int, error)
		WriteError(interface{})
		WriteErrorf(f string, args ...interface{})
	}

	// Prompter contains methods enabling user input
	Prompter interface {
		Prompt(p string, options ...string) (string, error)
		Confirm(p string, d bool) (bool, error)
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

	// DefaultTerminal implementation of Terminal interface
	DefaultTerminal struct {
		out   Writer
		err   io.Writer
		in    Reader
		start time.Time
		spnr  *DefaultSpinner
	}

	// SpinnerStatus defines a spinner status message
	SpinnerStatus string

	contextType string
)

var (
	terminalContext contextType = "terminal"

	// the regex for reading from pipe
	yesRx = regexp.MustCompile("^(?i:y(?:es)?)$")
	noRx  = regexp.MustCompile("^(?i:n(?:o)?)$")
)

// Color returns a colorable terminal
func Color() *DefaultTerminal {
	wr := &colorWriter{
		Writer: colorable.NewColorableStdout(),
		fd:     os.Stdout.Fd(),
	}

	return New(wr, os.Stdin, colorable.NewColorableStderr())
}

// New returns a new terminal with the specified streams
func New(out Writer, in Reader, err io.Writer) *DefaultTerminal {
	t := DefaultTerminal{
		out:   out,
		err:   err,
		in:    in,
		start: time.Now(),
		spnr:  StandardSpinner(),
	}

	t.spnr.spinner.Writer = t.Error()

	return &t
}

// Printf writes a formatted message to the output stream
func (t *DefaultTerminal) Printf(f string, args ...interface{}) {
	if _, err := t.Write([]byte(fmt.Sprintf(f, args...))); err != nil {
		t.WriteError(err)
	}
}

// Writeln writes a line to the terminal
func (t *DefaultTerminal) Writeln(args ...interface{}) (int, error) {
	return fmt.Fprintln(t.out, args...)
}

func (t *DefaultTerminal) Write(v []byte) (n int, err error) {
	msg := string(v)
	return fmt.Fprint(t.out, msg)
}

// WriteErrorf writes a formatted message to the error stream
func (t *DefaultTerminal) WriteErrorf(f string, args ...interface{}) {
	fmt.Fprintf(t.err, f, args...)
}

// WriteError write a message to the error stream
func (t *DefaultTerminal) WriteError(v interface{}) {
	fmt.Fprintf(t.err, fmt.Sprint(v))
}

// Error return the error writer
func (t *DefaultTerminal) Error() io.Writer {
	return t.err
}

// Prompt prompts the use for an open or multiple choice anwswer
func (t *DefaultTerminal) Prompt(p string, options ...string) (string, error) {
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

	err := survey.Ask([]*survey.Question{&q}, &answers, survey.WithStdio(t.in, t.out, t.err))
	if err != nil {
		return "", err
	}

	return answers.Q, nil
}

// Confirm asks the user for a Y/n response, with a default
func (t *DefaultTerminal) Confirm(p string, def bool) (bool, error) {
	rval := def

	q := &survey.Confirm{
		Message: p,
		Default: def,
	}

	// there is a known issue https://github.com/AlecAivazis/survey/issues/394 with survey.AskOne(...) not able to handle
	// piped input data. eg: `yes n | akamai update`
	// workaround: check if input data is from pipe or from terminal before calling survey.AskOne(...)
	fi, err := os.Stdin.Stat()
	if err != nil {
		return def, err
	}
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		// data is from pipe
		reader := bufio.NewReader(os.Stdin)
		fmt.Println(q.Message)
		val, _, err := reader.ReadLine()
		if err != nil {
			return def, err
		}
		switch {
		case yesRx.Match(val):
			rval = true
		case noRx.Match(val):
			rval = false
		case string(val) == "":
			rval = def
		}
	} else {
		// data is from terminal
		err = survey.AskOne(q, &rval, survey.WithStdio(t.in, t.out, t.err))
	}

	return rval, err
}

// IsTTY returns true if the terminal is a valid tty
func (t *DefaultTerminal) IsTTY() bool {
	return isatty.IsTerminal(t.out.Fd()) || isatty.IsCygwinTerminal(t.out.Fd())
}

// Spinner returns the terminal spinner
func (t *DefaultTerminal) Spinner() Spinner {
	return t.spnr
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
func Context(ctx context.Context, term Terminal) context.Context {
	return context.WithValue(ctx, terminalContext, term)
}

// Get gets the terminal from the context
func Get(ctx context.Context) Terminal {
	t, ok := ctx.Value(terminalContext).(Terminal)
	if !ok {
		panic(errors.New("context does not have a DefaultTerminal"))
	}

	return t
}

func (w *colorWriter) Fd() uintptr {
	return w.fd
}

// ShowBanner displays welcome banner
func ShowBanner(ctx context.Context) {
	term := Get(ctx)

	if _, err := term.Writeln(); err != nil {
		term.WriteError(err.Error())
	}
	term.Printf(color.ReverseVideoString(strings.Repeat(" ", 60) + "\n"))
	title := "Welcome to Akamai CLI v" + version.Version
	ws := strings.Repeat(" ", 16)
	term.Printf(color.ReverseVideoString(ws + title + ws + "\n"))
	term.Printf(color.ReverseVideoString(strings.Repeat(" ", 60) + "\n"))
	if _, err := term.Writeln(); err != nil {
		term.WriteError(err.Error())
	}
}
