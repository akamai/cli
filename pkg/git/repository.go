package git

import (
	"context"
	"errors"
	"fmt"

	"github.com/akamai/cli/pkg/terminal"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

const (
	// DefaultRemoteName will provide default origin.
	DefaultRemoteName = git.DefaultRemoteName
)

var (
	// ErrPackageNotAvailable is an error for handling incorrect package
	ErrPackageNotAvailable = errors.New("package is not available. Supported packages can be found here: https://techdocs.akamai.com/home/page/products-tools-a-z")
)

// Repository interface.
type Repository interface {
	Open(path string) error
	Clone(ctx context.Context, path, repo string, isBare bool, progress terminal.Spinner) error
	Pull(ctx context.Context, worktree *git.Worktree) error
	Head() (*plumbing.Reference, error)
	Worktree() (*git.Worktree, error)
	CommitObject(h plumbing.Hash) (*object.Commit, error)
	Reset(opts *git.ResetOptions) error
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
		return translateError(err, "Unable to clone repository: %w")
	}
	r.gitRepo = gitRepo
	return nil
}

func (r *repository) Pull(ctx context.Context, worktree *git.Worktree) error {
	return translateError(worktree.PullContext(ctx, &git.PullOptions{RemoteName: DefaultRemoteName}), "Unable to fetch updates (%w)")
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

func (r *repository) Reset(opts *git.ResetOptions) error {
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	if err := w.Reset(opts); err != nil {
		return fmt.Errorf("unable to perform `git reset`: %s", err.Error())
	}
	return nil
}

func translateError(err error, defaultErrorFormat string) error {
	if err == nil {
		return nil
	}
	if err == transport.ErrAuthenticationRequired {
		return ErrPackageNotAvailable
	}
	return fmt.Errorf(defaultErrorFormat, err)
}
