package packages

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// Mock LangManager interface
type Mock struct {
	mock.Mock
}

// Install mock
func (m *Mock) Install(_ context.Context, dir string, requirements LanguageRequirements, commands []string) error {
	args := m.Called(dir, requirements, commands)
	return args.Error(0)
}

// FindExec mock
func (m *Mock) FindExec(_ context.Context, requirements LanguageRequirements, cmdExec string) ([]string, error) {
	args := m.Called(requirements, cmdExec)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// FinishExecution mocks behavior of (*langManager) FinishExecution()
func (m *Mock) FinishExecution(_ context.Context, languageRequirements LanguageRequirements, dirName string) {
	m.Called(languageRequirements, dirName)
}

// PrepareExecution mocks behavior of (*langManager) PrepareExecution()
func (m *Mock) PrepareExecution(_ context.Context, languageRequirements LanguageRequirements, dirName string) error {
	args := m.Called(languageRequirements, dirName)
	return args.Error(0)
}

// GetShell mocks behavior of (*langManager) GetShell()
func (m *Mock) GetShell(goos string) (string, error) {
	args := m.Called(goos)
	return args.Get(0).(string), args.Error(0)
}

// GetOS mocks behavior of (*langManager) GetOS()
func (m *Mock) GetOS() string {
	m.Called()
	return "GetOS()"
}

// FileExists mocks behavior of (*langManager) FileExists()
func (m *Mock) FileExists(path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}
