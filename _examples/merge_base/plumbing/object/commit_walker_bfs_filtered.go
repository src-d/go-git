package object

import (
	"io"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

// NewFilterCommitIter returns a CommitIter that walks the commit history,
// starting at the passed commit and visiting its parents in Breadth-first order.
// The commits returned by the CommitIter will validate the passed CommitFilter.
// The history won't be transversed beyond a commit if isLimit is true for it.
// Each commit will be visited only once.
// If the commit history can not be traversed, or the Close() method is called,
// the CommitIter won't return more commits.
// If no isValid is passed, all ancestors of from commit will be valid.
// If no isLimit is limmit, all ancestors of all commits will be visited.
func NewFilterCommitIter(
	// REVIEWER: store argument wouldn't be needed if this were part of go-git/plumbing/object package
	store storer.EncodedObjectStorer,
	from *object.Commit,
	isValid *CommitFilter,
	isLimit *CommitFilter,
) object.CommitIter {
	var validFilter CommitFilter
	if isValid == nil {
		validFilter = func(_ *object.Commit) bool {
			return true
		}
	} else {
		validFilter = *isValid
	}

	var limitFilter CommitFilter
	if isLimit == nil {
		limitFilter = func(_ *object.Commit) bool {
			return false
		}
	} else {
		limitFilter = *isLimit
	}

	return &filterCommitIter{
		// REVIEWER: store wouldn't be needed if this were part of go-git/plumbing/object package
		store:   store,
		isValid: validFilter,
		isLimit: limitFilter,
		visited: map[plumbing.Hash]bool{},
		queue:   []*object.Commit{from},
	}
}

// CommitFilter returns a boolean for the passed Commit
type CommitFilter func(*object.Commit) bool

// filterCommitIter implments object.CommitIter
type filterCommitIter struct {
	// REVIEWER: store wouldn't be needed if this were part of go-git/plumbing/object package
	store   storer.EncodedObjectStorer
	isValid CommitFilter
	isLimit CommitFilter
	visited map[plumbing.Hash]bool
	queue   []*object.Commit
	lastErr error
}

// Next returns the next commit of the CommitIter.
// It will return io.EOF if there are no more commits to visit,
// or an error if the history could not be traversed.
func (w *filterCommitIter) Next() (*object.Commit, error) {
	var commit *object.Commit
	var err error
	for {
		commit, err = w.popNewFromQueue()
		if err != nil {
			return nil, w.close(err)
		}

		w.visited[commit.Hash] = true

		if !w.isLimit(commit) {
			// REVIEWER: first argument would be commit.s if this were part of object.Commit
			err = w.addToQueue(w.store, commit.ParentHashes...)
			if err != nil {
				return nil, w.close(err)
			}
		}

		if w.isValid(commit) {
			return commit, nil
		}
	}
}

// ForEach runs the passed callback over each Commit returned by the CommitIter
// until the callback returns an error or there is no more commits to traverse.
func (w *filterCommitIter) ForEach(cb func(*object.Commit) error) error {
	for {
		commit, err := w.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if err := cb(commit); err != nil {
			return err
		}
	}

	return nil
}

// Error returns the error that caused that the CommitIter is no longer returning commits
func (w *filterCommitIter) Error() error {
	return w.lastErr
}

// Close closes the CommitIter
func (w *filterCommitIter) Close() {
	w.visited = map[plumbing.Hash]bool{}
	w.queue = []*object.Commit{}
	w.isLimit = nil
	w.isValid = nil
}

// close closes the CommitIter with an error
func (w *filterCommitIter) close(err error) error {
	w.Close()
	w.lastErr = err
	return err
}

// popNewFromQueue returns the first new commit from the internal fifo queue, or
// an io.EOF error if the queue is empty
func (w *filterCommitIter) popNewFromQueue() (*object.Commit, error) {
	var first *object.Commit
	for {
		if len(w.queue) == 0 {
			if w.lastErr != nil {
				return nil, w.lastErr
			}

			return nil, io.EOF
		}

		first = w.queue[0]
		w.queue = w.queue[1:]
		if w.visited[first.Hash] {
			continue
		}

		return first, nil
	}
}

// addToQueue adds the passed commits to the internal fifo queue if they weren'r already seen
// or returns an error if the passed hashes could not be used to get valid commits
func (w *filterCommitIter) addToQueue(
	store storer.EncodedObjectStorer,
	hashes ...plumbing.Hash,
) error {
	for _, hash := range hashes {
		if w.visited[hash] {
			continue
		}

		commit, err := object.GetCommit(store, hash)
		if err != nil {
			return err
		}

		w.queue = append(w.queue, commit)
	}

	return nil
}
