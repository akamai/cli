package config

import (
	"context"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Save(ctx context.Context) error {
	args := m.Called()
	return args.Error(0)
}

func (m *Mock) Values() map[string]map[string]string {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]map[string]string)
}

func (m *Mock) GetValue(section string, key string) (string, bool) {
	args := m.Called(section, key)
	return args.String(0), args.Bool(1)
}

func (m *Mock) SetValue(section string, key string, value string) {
	m.Called(section, key, value)
}

func (m *Mock) UnsetValue(section string, key string) {
	m.Called(section, key)
}

func (m *Mock) ExportEnv(ctx context.Context) error {
	args := m.Called()
	return args.Error(0)
}
