package config

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// Mock impl of Config interface
type Mock struct {
	mock.Mock
}

// Save mock
func (m *Mock) Save(_ context.Context) error {
	args := m.Called()
	return args.Error(0)
}

// Values mock
func (m *Mock) Values() map[string]map[string]string {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]map[string]string)
}

// GetValue mock
func (m *Mock) GetValue(section string, key string) (string, bool) {
	args := m.Called(section, key)
	return args.String(0), args.Bool(1)
}

// SetValue mock
func (m *Mock) SetValue(section string, key string, value string) {
	m.Called(section, key, value)
}

// UnsetValue mock
func (m *Mock) UnsetValue(section string, key string) {
	m.Called(section, key)
}

// ExportEnv mock
func (m *Mock) ExportEnv(_ context.Context) error {
	args := m.Called()
	return args.Error(0)
}
