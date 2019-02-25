package repository

import (
	"fmt"

	obj "gopkg.in/src-d/go-git.v4/_examples/merge_base/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

// errIsReachable is thrown when first commit is an ancestor of the second
var errIsReachable = fmt.Errorf("first is reachable from second")

// MergeBase mimics the behavior of `git merge-base first second`, returning the
// best common ancestor of the two passed commits
// The best common ancestors can not be reached from other common ancestors
func MergeBase(
	// REVIEWER: store param wouldn't be needed if MergeBase were part of go-git/git.Repository
	store storer.EncodedObjectStorer,
	first *object.Commit,
	second *object.Commit,
) ([]*object.Commit, error) {

	secondHistory, err := ancestorsIndex(first, second)
	if err == errIsReachable {
		return []*object.Commit{first}, nil
	}

	if err != nil {
		return nil, err
	}

	var res []*object.Commit
	inSecondHistory := isInIndexCommitFilter(secondHistory)
	// REVIEWER: store argument wouldn't be needed if this were part of go-git/plumbing/object package
	resIter := obj.NewFilterCommitIter(store, first, &inSecondHistory, &inSecondHistory)
	err = resIter.ForEach(func(commit *object.Commit) error {
		res = append(res, commit)
		return nil
	})

	return Independents(res)
}

// IsAncestor returns true if the first commit is ancestor of the second one
// It returns an error if the history is not transversable
// It mimics the behavior of `git merge --is-ancestor first second`
func IsAncestor(
	first *object.Commit,
	second *object.Commit,
) (bool, error) {
	_, err := ancestorsIndex(first, second)
	if err == errIsReachable {
		return true, nil
	}

	return false, nil
}

// ancestorsIndex returns a map with the ancestors of the first commit if the
// second one is not one of them. It returns errIsReachable if the second one is
// ancestor, or another error if the history is not transversable
func ancestorsIndex(first, second *object.Commit) (map[plumbing.Hash]bool, error) {
	if first.Hash.String() == second.Hash.String() {
		return nil, errIsReachable
	}

	secondHistory := map[plumbing.Hash]bool{}
	secondIter := object.NewCommitIterBSF(second, nil, nil)
	err := secondIter.ForEach(func(commit *object.Commit) error {
		if commit.Hash == first.Hash {
			return errIsReachable
		}

		secondHistory[commit.Hash] = true
		return nil
	})

	if err == errIsReachable {
		return nil, errIsReachable
	}

	if err != nil {
		return nil, err
	}

	return secondHistory, nil
}

// Independents returns a subset of the passed commits, that are not reachable from any other
// It mimics the behavior of `git merge-base --independent commit...`
func Independents(commits []*object.Commit) ([]*object.Commit, error) {
	return independents(commits, 0)
}

func independents(commits []*object.Commit, start int) ([]*object.Commit, error) {
	if len(commits) == 1 {
		return commits, nil
	}

	res := commits
	for i := start; i < len(commits); i++ {
		from := commits[i]
		fromHistoryIter := object.NewCommitIterBSF(from, nil, nil)
		err := fromHistoryIter.ForEach(func(fromAncestor *object.Commit) error {
			for _, other := range commits {
				if from.Hash != other.Hash && fromAncestor.Hash == other.Hash {
					res = remove(res, other)
				}
			}

			if len(res) == 1 {
				return storer.ErrStop
			}

			return nil
		})

		if err != nil {
			return nil, err
		}

		if len(res) < len(commits) {
			return independents(res, start)
		}

	}

	return commits, nil
}

func remove(commits []*object.Commit, toDelete *object.Commit) []*object.Commit {
	var res []*object.Commit
	for _, commit := range commits {
		if toDelete.Hash != commit.Hash {
			res = append(res, commit)
		}
	}

	return res
}

// isInIndexCommitFilter returns a commitFilter that returns true
// if the commit is in the passed index.
func isInIndexCommitFilter(index map[plumbing.Hash]bool) obj.CommitFilter {
	return func(c *object.Commit) bool {
		return index[c.Hash]
	}
}
