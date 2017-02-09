package merkletrie

import (
	"bytes"
	"fmt"
	"io"

	"srcd.works/go-git.v4/utils/merkletrie/noder"
)

// Action values represent the kind of things a Change can represent:
// insertion, deletions or modifications of files.
type Action int

// The set of possible actions in a change.
const (
	Insert Action = iota
	Delete
	Modify
)

// String returns the action as a human readable text.
func (a Action) String() string {
	switch a {
	case Insert:
		return "Insert"
	case Delete:
		return "Delete"
	case Modify:
		return "Modify"
	default:
		panic(fmt.Sprintf("unsupported action: %d", a))
	}
}

// A Change value represent how a noder has change between to merkletries.
type Change struct {
	// The kind of the change.
	Action Action
	// The noder before the change or nil if it was inserted.
	From noder.Path
	// The noder after the change or nil if it was deleted.
	To noder.Path
}

// NewInsert returns a new Change representing the insertion of n.
func NewInsert(n noder.Path) Change {
	return Change{
		Action: Insert,
		To:     n,
	}
}

// NewDelete returns a new Change representing the deletion of n.
func NewDelete(n noder.Path) Change {
	return Change{
		Action: Delete,
		From:   n,
	}
}

// NewModify returns a new Change representing that a has been modified and
// it is now b.
func NewModify(a, b noder.Path) Change {
	return Change{
		Action: Modify,
		From:   a,
		To:     b,
	}
}

// String returns a single change in human readable form, using the
// format: '<' + action + space + path + '>'.  The contents of the file
// before or after the change are not included in this format.
//
// Example: inserting a file at the path a/b/c.txt will return "<Insert
// a/b/c.txt>".
func (c Change) String() string {
	var buf bytes.Buffer

	_, _ = buf.WriteRune('<')
	_, _ = buf.WriteString(c.Action.String())
	_, _ = buf.WriteRune(' ')
	switch c.Action {
	case Insert:
		_, _ = buf.WriteString(c.To.String())
	case Delete:
		_, _ = buf.WriteString(c.From.String())
	case Modify:
		_, _ = buf.WriteString(c.To.String())
	}
	_, _ = buf.WriteRune('>')

	return buf.String()
}

// Changes is a list of changes between to merkletries.
type Changes []Change

// NewChanges returns an empty list of changes.
func NewChanges() Changes {
	return Changes{}
}

// Add adds the change c to the list of changes.
func (l *Changes) Add(c Change) {
	*l = append(*l, c)
}

// AddRecursiveInsert adds the required changes to insert all the
// file-like noders found in root, recursively.
func (l *Changes) AddRecursiveInsert(root noder.Path) error {
	return l.addRecursive(root, NewInsert)
}

// AddRecursiveDelete adds the required changes to delete all the
// file-like noders found in root, recursively.
func (l *Changes) AddRecursiveDelete(root noder.Path) error {
	return l.addRecursive(root, NewDelete)
}

type noderToChangeFn func(noder.Path) Change // NewInsert or NewDelete

func (l *Changes) addRecursive(root noder.Path, ctor noderToChangeFn) error {
	if !root.IsDir() {
		l.Add(ctor(root))
		return nil
	}

	i, err := NewIterFromPath(root)
	if err != nil {
		return err
	}

	var current noder.Path
	for {
		if current, err = i.Step(); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if current.IsDir() {
			continue
		}
		l.Add(ctor(current))
	}

	return nil
}
