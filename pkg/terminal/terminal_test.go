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
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestWrite(t *testing.T) {
	out, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Remove(out.Name())) // clean up
	}()

	term := New(out, nil, DiscardWriter())

	term.Write([]byte(t.Name()))

	_, err = out.Seek(0, 0)
	require.NoError(t, err)
	data, err := ioutil.ReadAll(out)
	require.NoError(t, err)

	assert.Equal(t, t.Name(), string(data))
}

func TestPrintf(t *testing.T) {
	out, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Remove(out.Name())) // clean up
	}()

	term := New(out, nil, DiscardWriter())

	term.Printf("test: %s", "abc")

	_, err = out.Seek(0, 0)
	require.NoError(t, err)
	data, err := ioutil.ReadAll(out)
	require.NoError(t, err)

	assert.Equal(t, "test: abc", string(data))
}

func TestWriteln(t *testing.T) {
	out, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Remove(out.Name())) // clean up
	}()

	term := New(out, nil, DiscardWriter())

	term.Writeln(t.Name())

	_, err = out.Seek(0, 0)
	require.NoError(t, err)
	data, err := ioutil.ReadAll(out)
	require.NoError(t, err)

	assert.Equal(t, t.Name()+"\n", string(data))
}

func TestWriteError(t *testing.T) {
	out, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Remove(out.Name())) // clean up
	}()

	term := New(os.Stdin, os.Stdin, out)

	term.WriteError(t.Name())

	_, err = out.Seek(0, 0)
	require.NoError(t, err)

	data, err := ioutil.ReadAll(out)
	require.NoError(t, err)

	assert.Equal(t, t.Name(), string(data))
}

func TestWriteErrorf(t *testing.T) {
	out, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Remove(out.Name())) // clean up
	}()

	term := New(os.Stdin, os.Stdin, out)

	term.WriteErrorf("test error: %s", "abc")

	_, err = out.Seek(0, 0)
	require.NoError(t, err)

	data, err := ioutil.ReadAll(out)
	require.NoError(t, err)

	assert.Equal(t, "test error: abc", string(data))
}

func TestPrompt(t *testing.T) {
	content := []byte("Tom\r\n")
	in, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Remove(in.Name())) // clean up
	}()

	_, err = in.Write(content)
	require.NoError(t, err)
	_, err = in.Seek(0, 0)
	require.NoError(t, err)

	term := New(DiscardWriter(), in, DiscardWriter())

	name, err := term.Prompt("What is your name")
	require.NoError(t, err)

	assert.Equal(t, "Tom", name)
}

func TestPromptOptions(t *testing.T) {
	content := []byte("yellow\r\n")
	in, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Remove(in.Name())) // clean up
	}()
	_, err = in.Write(content)
	require.NoError(t, err)
	_, err = in.Seek(0, 0)
	require.NoError(t, err)

	term := New(DiscardWriter(), in, DiscardWriter())

	color, err := term.Prompt("What is your favorite color", "yellow", "red", "blue")
	require.NoError(t, err)

	assert.Equal(t, "yellow", color)
}

func TestConfirm(t *testing.T) {
	t.Log("we are unable to test Confirm method as underlying survey library uses RuneReader input for which cannot be mocked")
	t.Skip()
	content := []byte("y\r\n")
	in, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.Remove(in.Name())) // clean up
	}()

	_, err = in.Write(content)
	require.NoError(t, err)
	_, err = in.Seek(0, 0)
	require.NoError(t, err)

	term := New(DiscardWriter(), in, DiscardWriter())

	val, err := term.Confirm("Are you here", false)
	require.NoError(t, err)

	assert.Equal(t, true, val)
}

func TestGet(t *testing.T) {
	tests := map[string]struct {
		givenTerm   Terminal
		shouldPanic bool
	}{
		"terminal found in context": {
			givenTerm: Color(),
		},
		"terminal not in context": {
			givenTerm:   nil,
			shouldPanic: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := Context(context.Background(), test.givenTerm)
			if test.shouldPanic {
				assert.PanicsWithError(t, "context does not have a DefaultTerminal", func() {
					Get(ctx)
				})
				return
			}
			term := Get(ctx)
			assert.Equal(t, test.givenTerm, term)
		})
	}
}

func TestShowBanner(t *testing.T) {
	out, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)
	term := New(out, nil, DiscardWriter())
	ctx := Context(context.Background(), term)
	ShowBanner(ctx)
	_, err = out.Seek(0, 0)
	require.NoError(t, err)
	data, err := ioutil.ReadAll(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Welcome to Akamai CLI")
}
