package git

import (
	"context"

	"gopkg.in/src-d/go-git.v4"

	"github.com/akamai/cli/pkg/terminal"
)

const (
	// DefaultRemoteName will provide default origin.
	DefaultRemoteName = git.DefaultRemoteName
)

// Repository interface.
type Repository interface {
	Open(path string) (*git.Repository, error)
	Clone(ctx context.Context, path, repo string, isBare bool, progress terminal.Spinner, depth int) (*git.Repository, error)
	Pull(ctx context.Context, worktree *git.Worktree) error
}

type repository struct{}

// NewRepository will initialize new git integrations instance.
func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Open(path string) (*git.Repository, error) {
	return git.PlainOpen(path)
}

func (r *repository) Clone(ctx context.Context, path, repo string, isBare bool, progress terminal.Spinner, depth int) (*git.Repository, error) {
	return git.PlainCloneContext(ctx, path, isBare, &git.CloneOptions{
		URL:      repo,
		Progress: progress,
		Depth:    depth,
	})
}

func (r *repository) Pull(ctx context.Context, worktree *git.Worktree) error {
	return worktree.PullContext(ctx, &git.PullOptions{RemoteName: DefaultRemoteName})
}
