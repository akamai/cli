package terminal

import (
	"github.com/stretchr/testify/mock"
	"io"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Write(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *Mock) Printf(f string, args ...interface{}) {
	_ = m.Called(f, args)
}

func (m *Mock) Writeln(args ...interface{}) (int, error) {
	a := m.Called(args)
	return a.Int(0), a.Error(1)
}

func (m *Mock) WriteError(i interface{}) {
	_ = m.Called(i)
}

func (m *Mock) WriteErrorf(f string, args ...interface{}) {
	_ = m.Called(f, args)
}

func (m *Mock) Prompt(p string, options ...string) (string, error) {
	args := m.Called(p, options)
	return args.String(0), args.Error(1)
}

func (m *Mock) Confirm(p string, d bool) (bool, error) {
	args := m.Called(p, d)
	return args.Bool(0), args.Error(1)
}

func (m *Mock) Spinner() Spinner {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(Spinner)
}

func (m *Mock) Error() io.Writer {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(io.Writer)
}

func (m *Mock) IsTTY() bool {
	args := m.Called()
	return args.Bool(0)
}
