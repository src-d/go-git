package object

import (
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"io"
)

type commitFileIter struct {
	fileName      string
	sourceIter    CommitIter
	currentCommit *Commit
}

// NewCommitFileIterFromIter returns a commit iterator which performs diffTree between
// successive trees returned from the commit iterator from the argument. The purpose of this is
// to find the commits that explain how the files that match the path came to be.
func NewCommitFileIterFromIter(fileName string, commitIter CommitIter) CommitIter {
	iterator := new(commitFileIter)
	iterator.sourceIter = commitIter
	iterator.fileName = fileName
	return iterator
}

func (c *commitFileIter) Next() (*Commit, error) {
	var err error
	if c.currentCommit == nil {
		c.currentCommit, err = c.sourceIter.Next()
		if err != nil {
			return nil, err
		}
	}

	for {
		// Parent-commit can be nil if the current-commit is the initial commit
		parentCommit, parentCommitErr := c.sourceIter.Next()
		if parentCommitErr != nil {
			if parentCommitErr != io.EOF {
				err = parentCommitErr
				break
			}
			parentCommit = nil
		}

		// Fetch the trees of the current and parent commits
		currentTree, currTreeErr := c.currentCommit.Tree()
		if currTreeErr != nil {
			err = currTreeErr
			break
		}

		var parentTree *Tree
		if parentCommit != nil {
			var parentTreeErr error
			parentTree, parentTreeErr = parentCommit.Tree()
			if parentTreeErr != nil {
				err = parentTreeErr
				break
			}
		}

		// Find diff between current and parent trees
		changes, diffErr := DiffTree(currentTree, parentTree)
		if diffErr != nil {
			err = diffErr
			break
		}

		foundChangeForFile := false
		for _, change := range changes {
			if change.name() == c.fileName {
				foundChangeForFile = true
				break
			}
		}

		// Storing the current-commit in-case a change is found, and
		// Updating the current-commit for the next-iteration
		prevCommit := c.currentCommit
		c.currentCommit = parentCommit

		if foundChangeForFile == true {
			return prevCommit, nil
		}

		// If there are no more commits to be found, then return with EOF
		if parentCommit == nil {
			err = io.EOF
			break
		}
	}

	// Setting current-commit to nil to prevent unwanted states when errors are raised
	c.currentCommit = nil
	return nil, err
}

func (c *commitFileIter) ForEach(cb func(*Commit) error) error {
	for {
		commit, nextErr := c.Next()
		if nextErr != nil {
			return nextErr
		}
		err := cb(commit)
		if err == storer.ErrStop {
			return nil
		} else if err != nil {
			return err
		}
	}
}

func (c *commitFileIter) Close() {
	c.sourceIter.Close()
}
