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
	"time"

	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cheggaaa/pb/v3"
)

type (
	// Terminal defines a terminal abstration interface
	Terminal interface {
		io.Writer

		Writef(f string, args ...interface{})
		WriteError(interface{})
		WriteErrorf(f string, args ...interface{})
		Prompt(p string, options ...string) (string, error)
		Progress(max int) Progress
	}

	// Progress is an interface for progress bars
	Progress interface {
		Writer() io.Writer
		SetTotal(t int)
		Add(count int)
		End()
	}
	terminal struct {
		Out   *os.File
		Err   io.Writer
		In    *os.File
		start time.Time
	}

	progress struct {
		p *pb.ProgressBar
	}
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
func New(out, in *os.File, err io.Writer) Terminal {
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
	msg := fmt.Sprintf("[%s] %s", time.Now().Sub(t.start).Truncate(time.Second), v)
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

func (t terminal) Progress(max int) Progress {
	p := &progress{
		p: pb.StartNew(max),
	}

	p.p.Set(pb.Bytes, true)
	p.p.SetWriter(t.Out)

	return p
}

func (p progress) Writer() io.Writer {
	return p.p.NewProxyWriter(ioutil.Discard)
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
