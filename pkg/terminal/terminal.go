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
	"io/ioutil"
	"strings"
	"time"

	"os"

	spnr "github.com/briandowns/spinner"
	"github.com/fatih/color"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cheggaaa/pb/v3"
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

		// Progress creates a limited progress bar using the output stream
		Progress(max int) Progress

		// Spinner creates a spinner using the output stream
		Spinner() Spinner
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

	// Progress is an interface for progress bars
	Progress interface {
		// io.Writer is used to update progress status using byte based progress
		io.Writer

		// SetTotal sets the limit for the progress bar
		SetTotal(t int)

		// Add increments the progress status
		Add(count int)

		// End terminates the progress bar
		End()
	}
	progress struct {
		p *pb.ProgressBar
		w io.Writer
	}

	// Spinner defines a simple status spinner interface
	Spinner interface {
		// io.Writer is to be used to update the status suffix
		io.Writer

		// Start begins the spinner with the initial status prefix
		Start(f string, args ...interface{})

		// Stop terminates the spinner with the final status message
		Stop(status SpinnerStatus)
	}

	// SpinnerStatus defines a spinner status message
	SpinnerStatus string

	spinner struct {
		prefix string
		s      *spnr.Spinner
	}
)

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
		fmt.Println(err.Error())
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

func (t terminal) Progress(max int) Progress {
	p := &progress{
		p: pb.StartNew(max),
	}
	p.w = p.p.NewProxyWriter(ioutil.Discard)

	p.p.Set(pb.Bytes, true)
	p.p.SetWriter(t.Out)

	return p
}

func (p progress) Write(v []byte) (n int, err error) {
	return p.w.Write(v)
}

func (p progress) SetTotal(t int) {
	p.p.SetTotal(int64(t))
}

func (p progress) Add(s int) {
	p.p.Add(s)
}

func (p progress) End() {
	p.p.Finish()
}

func (t terminal) Spinner() Spinner {
	s := spnr.New(spnr.CharSets[33], 500*time.Millisecond)
	s.Writer = t

	return &spinner{
		s: s,
	}
}

func (s *spinner) Start(f string, args ...interface{}) {
	s.prefix = fmt.Sprintf(f, args...)
	s.s.Prefix = s.prefix + " "
	s.s.Start()
}

func (s *spinner) Stop(status SpinnerStatus) {
	s.s.Suffix = ""
	s.s.FinalMSG = s.prefix + " " + string(status)
	s.s.Stop()
}

func (s *spinner) Write(v []byte) (n int, err error) {
	s.s.Suffix = " " + strings.TrimSpace(string(v))
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
