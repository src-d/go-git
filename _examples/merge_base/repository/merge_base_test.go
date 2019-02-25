package repository

import (
	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v4/_examples/merge_base/testing"
)

/*
// TestCase history

* 6ecf0ef2c2dffb796033e5a02219af86ec6584e5 		// first
|
| * e8d3ffab552895c19b9fcf7aa264d277cde33881	// second
|/
* 918c48b83bd081e863dbe1b80f8998f058cd8294		// merge-base first second
|
* af2d6a6954d532f8ffb47615169c8fdf9d383a1a		// firstAncestor -> merge-base first firstAncestor
|
* 1669dce138d9b841a518c64b10914d88f5e488ea
|\
| * a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69
| |\
| | * b8e471f58bcbca63b07bda20e428190409c2db47	// beyondMerge -> merge-base first beyondMerge
| |/
* | 35e85108805c84807bc66a02d91535e1e24b38b9
|/
* b029517f6300c2da0f4b651b8642506cd6aaf45d
*/

// TestMergeBase validates a simple merge-base case: between first and second commits
func (s *repositorySuite) TestMergeBase(c *C) {
	hashes := []string{
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5", // first
		"e8d3ffab552895c19b9fcf7aa264d277cde33881", // second
	}
	commits, err := commitsFromHashes(s.repository, hashes)
	c.Assert(err, IsNil)

	commits, err = MergeBase(s.repository.Storer, commits[0], commits[1])
	c.Assert(err, IsNil)

	expected := []string{
		"918c48b83bd081e863dbe1b80f8998f058cd8294", // merge-base first second
	}

	testing.AssertCommits(c, commits, expected)
}

// TestMergeBaseSelf asserts that merge-base between a commit and self, is the same commit
func (s *repositorySuite) TestMergeBaseSelf(c *C) {
	hashes := []string{
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5", // first
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5", // first
	}
	commits, err := commitsFromHashes(s.repository, hashes)
	c.Assert(err, IsNil)

	commits, err = MergeBase(s.repository.Storer, commits[0], commits[1])
	c.Assert(err, IsNil)

	expected := []string{
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5", // first
	}

	testing.AssertCommits(c, commits, expected)
}

// TestMergeBaseAncestor asserts that merge-base between a commit and one ancestor, is the ancestor
func (s *repositorySuite) TestMergeBaseAncestor(c *C) {
	hashes := []string{
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5", // first
		"af2d6a6954d532f8ffb47615169c8fdf9d383a1a", // firstAncestor
	}
	commits, err := commitsFromHashes(s.repository, hashes)
	c.Assert(err, IsNil)

	commits, err = MergeBase(s.repository.Storer, commits[0], commits[1])
	c.Assert(err, IsNil)

	expected := []string{
		"af2d6a6954d532f8ffb47615169c8fdf9d383a1a", // firstAncestor
	}

	testing.AssertCommits(c, commits, expected)
}

// TestMergeBaseWithMerges validates a merge-base between first and an ancestor of first that is beyond a merge
func (s *repositorySuite) TestMergeBaseWithMerges(c *C) {
	hashes := []string{
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5", // first
		"b8e471f58bcbca63b07bda20e428190409c2db47", // beyondMerge
	}
	commits, err := commitsFromHashes(s.repository, hashes)
	c.Assert(err, IsNil)

	commits, err = MergeBase(s.repository.Storer, commits[0], commits[1])
	c.Assert(err, IsNil)

	expected := []string{
		"b8e471f58bcbca63b07bda20e428190409c2db47", // beyondMerge
	}

	testing.AssertCommits(c, commits, expected)
}
