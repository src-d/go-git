// Package revlist provides support to access the ancestors of commits, in a
// similar way as the git-rev-list command.
package revlist

import (
	"io"

	"srcd.works/go-git.v4/plumbing"
	"srcd.works/go-git.v4/plumbing/object"
)

// Objects applies a complementary set. It gets all the hashes from all
// the reachable objects from the given commits. Ignore param are object hashes
// that we want to ignore on the result. It is a list because is
// easier to interact with other porcelain elements, but internally it is
// converted to a map. All that objects must be accessible from the object
// storer.
func Objects(
	commits []*object.Commit,
	ignore []plumbing.Hash) ([]plumbing.Hash, error) {

	seen := hashListToSet(ignore)
	result := make(map[plumbing.Hash]bool)
	for _, c := range commits {
		err := reachableObjects(c, seen, func(h plumbing.Hash) {
			if !seen[h] {
				result[h] = true
				seen[h] = true
			}
		})

		if err != nil {
			return nil, err
		}
	}

	return hashSetToList(result), nil
}

// reachableObjects returns, using the callback function, all the reachable
// objects from the specified commit. To avoid to iterate over seen commits,
// if a commit hash is into the 'seen' set, we will not iterate all his trees
// and blobs objects.
func reachableObjects(
	commit *object.Commit,
	seen map[plumbing.Hash]bool,
	cb func(h plumbing.Hash)) error {
	return object.WalkCommitHistory(commit, func(commit *object.Commit) error {
		if seen[commit.Hash] {
			return nil
		}

		cb(commit.Hash)

		return iterateCommitTrees(seen, commit, cb)
	})
}

// iterateCommitTrees iterate all reachable trees from the given commit
func iterateCommitTrees(
	seen map[plumbing.Hash]bool,
	commit *object.Commit,
	cb func(h plumbing.Hash)) error {

	tree, err := commit.Tree()
	if err != nil {
		return err
	}

	if seen[tree.Hash] {
		return nil
	}

	cb(tree.Hash)

	treeWalker := object.NewTreeWalker(tree, true)

	for {
		_, e, err := treeWalker.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if seen[e.Hash] {
			continue
		}

		cb(e.Hash)
	}

	return nil
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
