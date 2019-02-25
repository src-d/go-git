package object

import (
	"testing"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git-fixtures.v3"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

/*
REVIEWER: This file contains mostly a bunch of copy-paste from:
 - plumbing/object/object_test.go
 - plumbing/object/commit_walker_test.go
If this package is moved into 'plumbing/object' package most of this could be deleted
*/

var _ = Suite(&filterCommitIterSuite{})

func Test(t *testing.T) { TestingT(t) }

type filterCommitIterSuite struct {
	baseTestSuite
}

type baseTestSuite struct {
	fixtures.Suite
	storer  storer.EncodedObjectStorer
	fixture *fixtures.Fixture
}

func (s *baseTestSuite) SetUpSuite(c *C) {
	s.Suite.SetUpSuite(c)
	s.fixture = fixtures.Basic().One()
	storer := filesystem.NewStorage(s.fixture.DotGit(), cache.NewObjectLRUDefault())
	s.storer = storer
}

func (s *baseTestSuite) commit(c *C, h plumbing.Hash) *object.Commit {
	commit, err := object.GetCommit(s.storer, h)
	c.Assert(err, IsNil)
	return commit
}

func commitsFromIter(iter object.CommitIter) ([]*object.Commit, error) {
	var commits []*object.Commit
	err := iter.ForEach(func(c *object.Commit) error {
		commits = append(commits, c)
		return nil
	})

	return commits, err
}
