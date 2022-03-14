package commands

import "github.com/stretchr/testify/mock"

// MockCmd is used to track activity on command.Command
type MockCmd struct {
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
