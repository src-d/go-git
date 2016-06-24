package index

import (
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/idxfile"
	"gopkg.in/src-d/go-git.v3/formats/packfile"
)

// Index is a database of objects and their offset in a packfile.
// Objects are identified by their hash.
type Index map[core.Hash]int64

// NewFrompackfile returns a new index from a packfile reader.
func NewFromPackfile(r io.Reader) (Index, error) {
	d := packfile.NewDecoder(r)
	hasesOffsets, err := d.HasesOffsets()
	if err != nil {
		return nil, err
	}

	return Index(hasesOffsets), nil
}

// NewFromIdx returns a new index from an idx file reader.
func NewFromIdx(r io.Reader) (Index, error) {
	d := idxfile.NewDecoder(r)
	idx := &idxfile.Idxfile{}
	err := d.Decode(idx)
	if err != nil {
		return nil, err
	}

	ind := make(Index)
	for _, e := range idx.Entries {
		if _, ok := ind[e.Hash]; ok {
			return nil, fmt.Errorf("duplicated hash: %s", e.Hash)
		}
		ind[e.Hash] = int64(e.Offset)
	}

	return ind, nil
}

// Get returns the offset that an object has the packfile.
func (i Index) Get(h core.Hash) (int64, error) {
	offset, ok := i[h]
	if !ok {
		return 0, core.ErrObjectNotFound
	}

	return offset, nil
}
