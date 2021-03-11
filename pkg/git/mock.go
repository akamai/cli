package git

import (
	"context"
	"github.com/akamai/cli/pkg/terminal"
	"github.com/stretchr/testify/mock"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// Mock impl of Repository interface
type Mock struct {
	mock.Mock
}

// Open mock
func (m *Mock) Open(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// Clone mock
func (m *Mock) Clone(_ context.Context, path, repo string, isBare bool, progress terminal.Spinner) error {
	args := m.Called(path, repo, isBare, progress)
	return args.Error(0)
}

// Pull mock
func (m *Mock) Pull(_ context.Context, worktree *git.Worktree) error {
	args := m.Called(worktree)
	return args.Error(0)
}

// Head mock
func (m *Mock) Head() (*plumbing.Reference, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*plumbing.Reference), args.Error(1)
}

// Worktree mock
func (m *Mock) Worktree() (*git.Worktree, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*git.Worktree), args.Error(1)
}

// CommitObject mock
func (m *Mock) CommitObject(h plumbing.Hash) (*object.Commit, error) {
	args := m.Called(h)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*object.Commit), args.Error(1)
}
