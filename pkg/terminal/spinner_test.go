package terminal

import (
	"bytes"
	"fmt"
	spnr "github.com/briandowns/spinner"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	wr := bytes.Buffer{}
	s := DefaultSpinner{
		spinner: spnr.New(spnr.CharSets[26], 1*time.Minute, spnr.WithWriter(&wr)),
	}
	s.Start("spinner %s", "test")
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		s.spinner.Lock()
		if wr.Len() > 0 {
			assert.Contains(t, wr.String(), "spinner test .")
			return
		}
		s.spinner.Unlock()
	}
	t.Fatal("no input on writer")
}

func TestStop(t *testing.T) {
	tests := map[string]struct {
		spinnerStatus SpinnerStatus
		expected      string
	}{
		"stop spinner OK": {
			spinnerStatus: SpinnerStatusOK,
			expected:      "... [OK]",
		},
		"stop spinner WARN OK": {
			spinnerStatus: SpinnerStatusWarnOK,
			expected:      "... [OK]",
		},
		"stop spinner WARN": {
			spinnerStatus: SpinnerStatusWarn,
			expected:      "... [WARN]",
		},
		"stop spinner FAIL": {
			spinnerStatus: SpinnerStatusFail,
			expected:      "... [FAIL]",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			wr := bytes.Buffer{}
			s := DefaultSpinner{
				spinner: spnr.New(spnr.CharSets[26], 1*time.Minute, spnr.WithWriter(&wr)),
			}
			s.Start("spinner %s", "test")
			s.Stop(test.spinnerStatus)
			assert.Contains(t, wr.String(), fmt.Sprintf("spinner test %s", test.expected))
		})
	}
}

func TestOK(t *testing.T) {
	wr := bytes.Buffer{}
	s := DefaultSpinner{
		spinner: spnr.New(spnr.CharSets[26], 1*time.Minute, spnr.WithWriter(&wr)),
	}
	s.Start("spinner %s", "test")
	s.OK()
	assert.Contains(t, wr.String(), fmt.Sprintf("spinner test ... [OK]"))
}

func TestWarn(t *testing.T) {
	wr := bytes.Buffer{}
	s := DefaultSpinner{
		spinner: spnr.New(spnr.CharSets[26], 1*time.Minute, spnr.WithWriter(&wr)),
	}
	s.Start("spinner %s", "test")
	s.Warn()
	assert.Contains(t, wr.String(), fmt.Sprintf("spinner test ... [WARN]"))
}

func TestWarnOK(t *testing.T) {
	wr := bytes.Buffer{}
	s := DefaultSpinner{
		spinner: spnr.New(spnr.CharSets[26], 1*time.Minute, spnr.WithWriter(&wr)),
	}
	s.Start("spinner %s", "test")
	s.WarnOK()
	assert.Contains(t, wr.String(), fmt.Sprintf("spinner test ... [OK]"))
}

func TestFail(t *testing.T) {
	wr := bytes.Buffer{}
	s := DefaultSpinner{
		spinner: spnr.New(spnr.CharSets[26], 1*time.Minute, spnr.WithWriter(&wr)),
	}
	s.Start("spinner %s", "test")
	s.Fail()
	assert.Contains(t, wr.String(), fmt.Sprintf("spinner test ... [FAIL]"))
}

func TestSpinnerWrite(t *testing.T) {
	wr := bytes.Buffer{}
	s := DefaultSpinner{
		spinner: spnr.New(spnr.CharSets[26], 1*time.Minute, spnr.WithWriter(&wr)),
	}
	l, err := s.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, l)
	assert.Equal(t, " test", s.spinner.Suffix)
}
