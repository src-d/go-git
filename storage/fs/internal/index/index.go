package index

import (
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/idxfile"
)

// Index is a database of the hashes of the objects and their offsets in
// the packfile.
type Index map[core.Hash]int64

// NewFromPackfile returns a new index from a packfile file reader.
func NewFromPackfile(packfile io.Reader) (Index, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// NewFromIdx returns a new index from an idx file reader.
func NewFromIdx(r io.Reader) (Index, error) {
	d := idxfile.NewDecoder(r)
	idx := &idxfile.Idxfile{}
	err := d.Decode(idx)
	if err != nil {
		return nil, err
	}

	result := make(Index)
	for _, entry := range idx.Entries {
		if _, ok := result[entry.Hash]; ok {
			return nil, fmt.Errorf("duplicated hash: %s", entry.Hash)
		}
		result[entry.Hash] = int64(entry.Offset)
	}

	return result, nil
}

// Get returns the offset of an object in the packfile.
func (i Index) Get(h core.Hash) (int64, error) {
	offset, ok := i[h]
	if !ok {
		return 0, core.ErrObjectNotFound
	}

	return offset, nil
}
