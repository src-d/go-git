package git

import (
	"io"

	"gopkg.in/src-d/go-git.v4/plumbing"
)

// RevListObjects gets all the hashes from all the reachable objects from the
// given commits. Ignore hashes are objects that you don't want back into the
// result. All that objects must be accessible from the Repository.
func RevListObjects(
	commits []*Commit,
	ignore []plumbing.Hash,
	r *Repository) ([]plumbing.Hash, error) {

	seen := hashListToSet(ignore)
	result := make(map[plumbing.Hash]bool)
	for _, c := range commits {
		err := iterateAll(c, seen, func(h plumbing.Hash) error {
			if !seen[h] {
				result[h] = true
				seen[h] = true
			}

			return nil
		}, r)

		if err != nil {
			return nil, err
		}
	}

	return hashSetToList(result), nil
}

func hashSetToList(hashes map[plumbing.Hash]bool) []plumbing.Hash {
	var result []plumbing.Hash
	for key := range hashes {
		result = append(result, key)
	}

	return result
}

func hashListToSet(hashes []plumbing.Hash) map[plumbing.Hash]bool {
	result := make(map[plumbing.Hash]bool)
	for _, h := range hashes {
		result[h] = true
	}

	return result
}

func iterateCommits(commit *Commit, cb func(c *Commit) error) error {
	if err := cb(commit); err != nil {
		return err
	}

	return WalkCommitHistory(commit, func(c *Commit) error {
		return cb(c)
	})
}

func iterateCommitTrees(
	commit *Commit,
	cb func(h plumbing.Hash) error,
	repository *Repository) error {

	tree, err := commit.Tree()
	if err != nil {
		return err
	}
	if err := cb(tree.Hash); err != nil {
		return err
	}

	treeWalker := NewTreeWalker(repository, tree, true)

	for {
		_, e, err := treeWalker.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := cb(e.Hash); err != nil {
			return err
		}
	}

	return nil
}

func iterateAll(
	commit *Commit,
	seen map[plumbing.Hash]bool,
	cb func(h plumbing.Hash) error,
	r *Repository) error {

	return iterateCommits(commit, func(commit *Commit) error {
		if seen[commit.Hash] {
			return nil
		}

		if err := cb(commit.Hash); err != nil {
			return err
		}

		return iterateCommitTrees(commit, func(h plumbing.Hash) error {
			return cb(h)
		}, r)
	})
}
