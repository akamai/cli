package commands

import "github.com/stretchr/testify/mock"

// MockCmd is used to track activity on command.Command
type MockCmd struct {
	mock.Mock
}

// mockPackageReader mocks the package reader
type mockPackageReader struct {
	mock.Mock
}

// String mimics the behavior or (*Command) String()
func (c *MockCmd) String() string {
	return "MockCmd"
}

// Run mimics the behavior of (*Command) Run()
func (c *MockCmd) Run() error {
	args := c.Called()
	return args.Error(0)
}

// readPackage() mimics the behavior of (*packageReader) readPackage() method
func (m *mockPackageReader) readPackage() (*packageList, error) {
	args := m.Called()
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*packageList), args.Error(1)
}
