package testing

import (
	"gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func AssertCommits(c *check.C, commits []*object.Commit, hashes []string) {
	c.Assert(commits, check.HasLen, len(hashes))
	for i, commit := range commits {
		c.Assert(hashes[i], check.Equals, commit.Hash.String())
	}
}

