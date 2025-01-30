package git

import (
	"context"

	"github.com/akamai/cli/v2/pkg/terminal"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/mock"
)

type (
	// MockRepo impl of Repository interface
	MockRepo struct {
		mock.Mock
	}
)

// Open mock
func (m *MockRepo) Open(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// Clone mock
func (m *MockRepo) Clone(_ context.Context, path, repo string, isBare bool, progress terminal.Spinner) error {
	args := m.Called(path, repo, isBare, progress)
	return args.Error(0)
}

// Pull mock
func (m *MockRepo) Pull(_ context.Context, worktree *git.Worktree) error {
	args := m.Called(worktree)
	return args.Error(0)
}

// Head mock
func (m *MockRepo) Head() (*plumbing.Reference, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plumbing.Reference), args.Error(1)
}

// Worktree mock
func (m *MockRepo) Worktree() (*git.Worktree, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*git.Worktree), args.Error(1)
}

// CommitObject mock
func (m *MockRepo) CommitObject(h plumbing.Hash) (*object.Commit, error) {
	args := m.Called(h)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*object.Commit), args.Error(1)
}

// Reset mock
func (m *MockRepo) Reset(opts *git.ResetOptions) error {
	args := m.Called(opts)
	return args.Error(0)
}
