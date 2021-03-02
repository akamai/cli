package terminal

import (
	"github.com/stretchr/testify/mock"
	"io"
)

// Mock terminal
type Mock struct {
	mock.Mock
}

// Write mock implementation
func (m *Mock) Write(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

// Printf mock implementation
func (m *Mock) Printf(f string, args ...interface{}) {
	_ = m.Called(f, args)
}

// Print mock implementation
func (m *Mock) Print(f string) {
	_ = m.Called(f)
}

// Writeln mock implementation
func (m *Mock) Writeln(args ...interface{}) (int, error) {
	a := m.Called(args)
	return a.Int(0), a.Error(1)
}

// WriteError mock implementation
func (m *Mock) WriteError(i interface{}) {
	_ = m.Called(i)
}

// WriteErrorf mock implementation
func (m *Mock) WriteErrorf(f string, args ...interface{}) {
	_ = m.Called(f, args)
}

// Prompt mock implementation
func (m *Mock) Prompt(p string, options ...string) (string, error) {
	args := m.Called(p, options)
	return args.String(0), args.Error(1)
}

// Confirm mock implementation
func (m *Mock) Confirm(p string, d bool) (bool, error) {
	args := m.Called(p, d)
	return args.Bool(0), args.Error(1)
}

// Spinner mock implementation
func (m *Mock) Spinner() Spinner {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(Spinner)
}

// Error mock implementation
func (m *Mock) Error() io.Writer {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(io.Writer)
}

// IsTTY mock implementation
func (m *Mock) IsTTY() bool {
	args := m.Called()
	return args.Bool(0)
}

// Start mock
func (m *Mock) Start(f string, args ...interface{}) {
	_ = m.Called(f, args)
}

// Stop mock
func (m *Mock) Stop(status SpinnerStatus) {
	_ = m.Called(status)
}

// OK mock
func (m *Mock) OK() {
	m.Called()
}

// WarnOK mock
func (m *Mock) WarnOK() {
	m.Called()
}

// Warn mock
func (m *Mock) Warn() {
	m.Called()
}

// Fail mock
func (m *Mock) Fail() {
	m.Called()
}
