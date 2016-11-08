package git

import (
	"io"

	"gopkg.in/src-d/go-git.v4/core"
)

// WalkCommitHistory walks the commit history
func WalkCommitHistory(c *Commit, cb func(*Commit) error) error {
	seen := map[core.Hash]bool{c.Hash: true}
	queue := []*Commit{c}

	for {
		if len(queue) == 0 {
			return nil
		}

		commit := queue[0]
		queue = queue[1:]

		if err := cb(commit); err != nil {
			return err
		}

		iter := commit.Parents()
		for {
			parent, err := iter.Next()
			if err == io.EOF {
				break
			}

			if err != nil {
				return err
			}

			if seen[parent.Hash] {
				continue
			}

			seen[parent.Hash] = true
			queue = append(queue, parent)
		}
	}
}
