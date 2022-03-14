package git

import (
	"context"
	"fmt"

	"github.com/akamai/cli/pkg/terminal"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

const (
	// DefaultRemoteName will provide default origin.
	DefaultRemoteName = git.DefaultRemoteName
)

// Repository interface.
type Repository interface {
	Open(path string) error
	Clone(ctx context.Context, path, repo string, isBare bool, progress terminal.Spinner) error
	Pull(ctx context.Context, worktree *git.Worktree) error
	Head() (*plumbing.Reference, error)
	Worktree() (*git.Worktree, error)
	CommitObject(h plumbing.Hash) (*object.Commit, error)
}

type repository struct {
	gitRepo *git.Repository
}

// NewRepository will initialize new git integrations instance.
func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Open(path string) error {
	gitRepo, err := git.PlainOpen(path)
	if err != nil {
		return err
	}
	r.gitRepo = gitRepo
	return nil
}

func (r *repository) Clone(ctx context.Context, path, repo string, isBare bool, progress terminal.Spinner) error {
	gitRepo, err := git.PlainCloneContext(ctx, path, isBare, &git.CloneOptions{
		URL:      repo,
		Progress: progress,
	})
	if err != nil {
		return err
	}
	r.gitRepo = gitRepo
	return nil
}

func (r *repository) Pull(ctx context.Context, worktree *git.Worktree) error {
	return worktree.PullContext(ctx, &git.PullOptions{RemoteName: DefaultRemoteName})
}

func (r *repository) Head() (*plumbing.Reference, error) {
	if r.gitRepo == nil {
		return nil, fmt.Errorf("repository is not yet initialized")
	}
	return r.gitRepo.Head()
}

func (r *repository) Worktree() (*git.Worktree, error) {
	if r.gitRepo == nil {
		return nil, fmt.Errorf("repository is not yet initialized")
	}
	return r.gitRepo.Worktree()
}

func (r *repository) CommitObject(h plumbing.Hash) (*object.Commit, error) {
	if r.gitRepo == nil {
		return nil, fmt.Errorf("repository is not yet initialized")
	}
	return r.gitRepo.CommitObject(h)
}
