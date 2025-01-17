package terminal

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	spnr "github.com/briandowns/spinner"
	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {
	s := DefaultSpinner{
		spinner: spnr.New(spnr.CharSets[26], 1*time.Minute),
	}
	t.Cleanup(func() {
		s.spinner.Stop()
	})

	s.Start("spinner %s", "test")

	s.spinner.Lock()
	assert.Contains(t, s.spinner.Prefix, "spinner test")
	s.spinner.Unlock()
}

func TestStop(t *testing.T) {
	tests := map[string]struct {
		spinnerStatus SpinnerStatus
		expected      string
	}{
		"stop spinner OK": {
			spinnerStatus: SpinnerStatusOK,
			expected:      "[OK]",
		},
		"stop spinner WARN OK": {
			spinnerStatus: SpinnerStatusWarnOK,
			expected:      "[OK]",
		},
		"stop spinner WARN": {
			spinnerStatus: SpinnerStatusWarn,
			expected:      "[WARN]",
		},
		"stop spinner FAIL": {
			spinnerStatus: SpinnerStatusFail,
			expected:      "[FAIL]",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			s := DefaultSpinner{
				spinner: spnr.New(spnr.CharSets[26], 1*time.Minute),
			}
			s.Start("spinner %s", "test")
			s.Stop(test.spinnerStatus)
			assert.Contains(t, s.spinner.FinalMSG, fmt.Sprintf("spinner test %s", test.expected))
		})
	}
}

func TestOK(t *testing.T) {
	s := DefaultSpinner{
		spinner: spnr.New(spnr.CharSets[26], 1*time.Minute),
	}
	s.Start("spinner %s", "test")
	s.OK()
	assert.Contains(t, s.spinner.FinalMSG, "spinner test [OK]")
}

func TestWarn(t *testing.T) {
	s := DefaultSpinner{
		spinner: spnr.New(spnr.CharSets[26], 1*time.Minute),
	}
	s.Start("spinner %s", "test")
	s.Warn()
	assert.Contains(t, s.spinner.FinalMSG, "spinner test [WARN]")
}

func TestWarnOK(t *testing.T) {
	s := DefaultSpinner{
		spinner: spnr.New(spnr.CharSets[26], 1*time.Minute),
	}
	s.Start("spinner %s", "test")
	s.WarnOK()
	assert.Contains(t, s.spinner.FinalMSG, "spinner test [OK]")
}

func TestFail(t *testing.T) {
	s := DefaultSpinner{
		spinner: spnr.New(spnr.CharSets[26], 1*time.Minute),
	}
	s.Start("spinner %s", "test")
	s.Fail()
	assert.Contains(t, s.spinner.FinalMSG, "spinner test [FAIL]")
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
