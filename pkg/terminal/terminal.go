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

	"os"

	"github.com/AlecAivazis/survey/v2"
)

type (
	// Terminal defines a terminal abstration interface
	Terminal interface {
		Write(f string, args ...interface{})
		WriteError(f string, args ...interface{})
		Prompt(p string, options ...string) (string, error)
		ProgressBegin(max ...int)
		ProgressStep(s int)
		ProgressEnd()
	}

	terminal struct {
		Out *os.File
		Err io.Writer
		In  *os.File
	}
)

// Standard returns the standard terminal
func Standard() Terminal {
	return terminal{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}
}

// New returns a new terminal with the specifed streams
func New(out, in *os.File, err io.Writer) Terminal {
	return terminal{
		Out: out,
		Err: err,
		In:  in,
	}
}

func (t terminal) Write(f string, args ...interface{}) {
	t.Out.Write([]byte(fmt.Sprintf(f, args...)))
}

func (t terminal) WriteError(f string, args ...interface{}) {
	t.Err.Write([]byte(fmt.Sprintf(f, args...)))
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

func (t terminal) ProgressBegin(max ...int) {

}

func (t terminal) ProgressStep(s int) {

}

func (t terminal) ProgressEnd() {

}
