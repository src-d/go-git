package storer

import "gopkg.in/src-d/go-git.v4/plumbing"

// ShallowStorer storage of shallow commits, meaning that it does not have the
// parents of a commit (explanation from git documentation)
type ShallowStorer interface {
	SetShallow([]plumbing.Hash) error
	Shallow() ([]plumbing.Hash, error)
}
