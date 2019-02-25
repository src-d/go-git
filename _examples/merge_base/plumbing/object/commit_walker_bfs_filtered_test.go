package object

import (
	. "gopkg.in/check.v1"

	"gopkg.in/src-d/go-git.v4/_examples/merge_base/testing"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

/*
// TestCase history

* 6ecf0ef2c2dffb796033e5a02219af86ec6584e5 <- HEAD
|
| * e8d3ffab552895c19b9fcf7aa264d277cde33881
|/
* 918c48b83bd081e863dbe1b80f8998f058cd8294
|
* af2d6a6954d532f8ffb47615169c8fdf9d383a1a
|
* 1669dce138d9b841a518c64b10914d88f5e488ea
|\
| * a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69	// isLimit
| |\
| | * b8e471f58bcbca63b07bda20e428190409c2db47  // ignored if isLimit is passed
| |/
* | 35e85108805c84807bc66a02d91535e1e24b38b9	// isValid; ignored if passed as !isValid
|/
* b029517f6300c2da0f4b651b8642506cd6aaf45d
*/

func validIfCommit(ignored plumbing.Hash) CommitFilter {
	return func(c *object.Commit) bool {
		if c.Hash == ignored {
			return true
		}

		return false
	}
}

func not(filter CommitFilter) CommitFilter {
	return func(c *object.Commit) bool {
		return !filter(c)
	}
}

// TestFilterCommitIter asserts that FilterCommitIter returns all commits from
// history, but e8d3ffab552895c19b9fcf7aa264d277cde33881, that is not reachable
// from HEAD
func (s *filterCommitIterSuite) TestFilterCommitIter(c *C) {
	commit := s.commit(c, s.fixture.Head)

	commits, err := commitsFromIter(NewFilterCommitIter(s.storer, commit, nil, nil))
	c.Assert(err, IsNil)

	expected := []string{
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5",
		"918c48b83bd081e863dbe1b80f8998f058cd8294",
		"af2d6a6954d532f8ffb47615169c8fdf9d383a1a",
		"1669dce138d9b841a518c64b10914d88f5e488ea",
		"35e85108805c84807bc66a02d91535e1e24b38b9",
		"a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69",
		"b029517f6300c2da0f4b651b8642506cd6aaf45d",
		"b8e471f58bcbca63b07bda20e428190409c2db47",
	}

	testing.AssertCommits(c, commits, expected)
}

// TestFilterCommitIterWithValid asserts that FilterCommitIter returns only commits
// that matches the passed isValid filter; in this testcase, it was filtered out
// all commits but one from history
func (s *filterCommitIterSuite) TestFilterCommitIterWithValid(c *C) {
	commit := s.commit(c, s.fixture.Head)

	validIf := validIfCommit(plumbing.NewHash("35e85108805c84807bc66a02d91535e1e24b38b9"))
	commits, err := commitsFromIter(NewFilterCommitIter(s.storer, commit, &validIf, nil))
	c.Assert(err, IsNil)

	expected := []string{
		"35e85108805c84807bc66a02d91535e1e24b38b9",
	}

	testing.AssertCommits(c, commits, expected)
}

// that matches the passed isValid filter; in this testcase, it was filtered out
// only one commit from history
func (s *filterCommitIterSuite) TestFilterCommitIterWithInvalid(c *C) {
	commit := s.commit(c, s.fixture.Head)

	validIf := validIfCommit(plumbing.NewHash("35e85108805c84807bc66a02d91535e1e24b38b9"))
	validIfNot := not(validIf)
	commits, err := commitsFromIter(NewFilterCommitIter(s.storer, commit, &validIfNot, nil))
	c.Assert(err, IsNil)

	expected := []string{
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5",
		"918c48b83bd081e863dbe1b80f8998f058cd8294",
		"af2d6a6954d532f8ffb47615169c8fdf9d383a1a",
		"1669dce138d9b841a518c64b10914d88f5e488ea",
		"a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69",
		"b029517f6300c2da0f4b651b8642506cd6aaf45d",
		"b8e471f58bcbca63b07bda20e428190409c2db47",
	}

	testing.AssertCommits(c, commits, expected)
}

// TestFilterCommitIterWithNoValidCommits asserts that FilterCommitIter returns
// no commits if the passed isValid filter does not allow any commit
func (s *filterCommitIterSuite) TestFilterCommitIterWithNoValidCommits(c *C) {
	commit := s.commit(c, s.fixture.Head)

	validIf := validIfCommit(plumbing.NewHash("THIS_COMMIT_DOES_NOT_EXIST"))
	commits, err := commitsFromIter(NewFilterCommitIter(s.storer, commit, &validIf, nil))
	c.Assert(err, IsNil)
	c.Assert(commits, HasLen, 0)
}

// TestFilterCommitIterWithStopAt asserts that FilterCommitIter returns only commits
// are not beyond a isLimit filter
func (s *filterCommitIterSuite) TestFilterCommitIterWithStopAt(c *C) {
	commit := s.commit(c, s.fixture.Head)

	stopAtRule := validIfCommit(plumbing.NewHash("a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69"))
	commits, err := commitsFromIter(NewFilterCommitIter(s.storer, commit, nil, &stopAtRule))
	c.Assert(err, IsNil)

	expected := []string{
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5",
		"918c48b83bd081e863dbe1b80f8998f058cd8294",
		"af2d6a6954d532f8ffb47615169c8fdf9d383a1a",
		"1669dce138d9b841a518c64b10914d88f5e488ea",
		"35e85108805c84807bc66a02d91535e1e24b38b9",
		"a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69",
		"b029517f6300c2da0f4b651b8642506cd6aaf45d",
	}

	testing.AssertCommits(c, commits, expected)
}

// TestFilterCommitIterWithStopAt asserts that FilterCommitIter works properly
// with isValid and isLimit filters
func (s *filterCommitIterSuite) TestFilterCommitIterWithInvalidAndStopAt(c *C) {
	commit := s.commit(c, s.fixture.Head)

	stopAtRule := validIfCommit(plumbing.NewHash("a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69"))
	validIf := validIfCommit(plumbing.NewHash("35e85108805c84807bc66a02d91535e1e24b38b9"))
	validIfNot := not(validIf)
	commits, err := commitsFromIter(NewFilterCommitIter(s.storer, commit, &validIfNot, &stopAtRule))
	c.Assert(err, IsNil)

	expected := []string{
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5",
		"918c48b83bd081e863dbe1b80f8998f058cd8294",
		"af2d6a6954d532f8ffb47615169c8fdf9d383a1a",
		"1669dce138d9b841a518c64b10914d88f5e488ea",
		"a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69",
		"b029517f6300c2da0f4b651b8642506cd6aaf45d",
	}

	testing.AssertCommits(c, commits, expected)
}
