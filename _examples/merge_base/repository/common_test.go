package repository

import (
	"testing"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git-fixtures.v3"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

/*
REVIEWER: This file contains mostly a bunch of copy-paste from:
 - common_test.go
 - repository_test.go
If this package is moved into 'git' package most of this could be deleted
*/

var _ = Suite(&repositorySuite{})

func Test(t *testing.T) { TestingT(t) }

type repositorySuite struct {
	baseSuite
}

type baseSuite struct {
	fixtures.Suite
	repository *git.Repository
}

func (s *baseSuite) SetUpSuite(c *C) {
	s.Suite.SetUpSuite(c)
	s.buildBasicRepository(c)
}

func (s *baseSuite) TearDownSuite(c *C) {
	s.Suite.TearDownSuite(c)
}

func (s *baseSuite) buildBasicRepository(c *C) {
	f := fixtures.Basic().One()
	s.repository = newRepository(f)
}

// NewRepository returns a new repository using the .git folder, if the fixture
// is tagged as worktree the filesystem from fixture is used, otherwise a new
// memfs filesystem is used as worktree.
func newRepository(f *fixtures.Fixture) *git.Repository {
	var worktree, dotgit billy.Filesystem
	if f.Is("worktree") {
		r, err := git.PlainOpen(f.Worktree().Root())
		if err != nil {
			panic(err)
		}

		return r
	}

	dotgit = f.DotGit()
	worktree = memfs.New()

	st := filesystem.NewStorage(dotgit, cache.NewObjectLRUDefault())

	r, err := git.Open(st, worktree)

	if err != nil {
		panic(err)
	}

	return r
}

func commitsFromHashes(repo *git.Repository, hashes []string) ([]*object.Commit, error) {
	var commits []*object.Commit
	for _, hash := range hashes {
		commit, err := repo.CommitObject(plumbing.NewHash(hash))
		if err != nil {
			return nil, err
		}

		commits = append(commits, commit)
	}

	return commits, nil
}
