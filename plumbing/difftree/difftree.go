package difftree

import (
	"bytes"
	"fmt"
	"io"

	"srcd.works/go-git.v4/plumbing/object"
	"srcd.works/go-git.v4/utils/merkletrie"
	"srcd.works/go-git.v4/utils/merkletrie/noder"
)

func DiffTree(a, b *object.Tree) ([]*Change, error) {
	if a == b {
		return Changes{}, nil
	}

	// TODO remove this by returning empty noders in newTreeNoderFromTree,
	// so we can delete newWithEmpty.
	if a == nil || b == nil {
		return newWithEmpty(a, b)
	}

	from := newTreeNoder(a)
	to := newTreeNoder(b)

	hashEqual := func(a, b noder.Hasher) bool {
		return bytes.Equal(a.Hash(), b.Hash())
	}

	merkletrieChanges, err := merkletrie.DiffTree(from, to, hashEqual)
	if err != nil {
		return nil, err
	}

	return newChanges(merkletrieChanges)
}

func newWithEmpty(a, b *object.Tree) (Changes, error) {
	changes := Changes{}

	var action merkletrie.Action
	var tree *object.Tree
	if a == nil {
		action = merkletrie.Insert
		tree = b
	} else {
		action = merkletrie.Delete
		tree = a
	}

	w := object.NewTreeWalker(tree, true)
	defer w.Close()

	for {
		path, entry, err := w.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("cannot get next file: %s", err)
		}

		if entry.Mode.IsDir() {
			continue
		}

		c := &Change{}

		if action == merkletrie.Insert {
			c.To.Name = path
			c.To.TreeEntry = entry
			c.To.Tree = tree
		} else {
			c.From.Name = path
			c.From.TreeEntry = entry
			c.From.Tree = tree
		}

		changes = append(changes, c)
	}

	return changes, nil
}
