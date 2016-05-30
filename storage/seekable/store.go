package seekable

import (
	"errors"
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/packfile"
	"gopkg.in/src-d/go-git.v3/storage/memory"
	"gopkg.in/src-d/go-git.v3/storage/seekable/internal/index"
)

// ObjectStorage is an implementation of core.ObjectStorage for seekable
// packfiles.
//
// The objects in the packfile are not stored in memory, instead
// every Get call will access the packfile, with some help from
// the packfile index and the fact that is seekable to speed
// things up.
//
// This means the memory footprint of this storage is much smaller
// than a memory.ObjectStorage, but it will also be probably slower.
type ObjectStorage struct {
	packfile io.ReadSeeker
	index    index.Index
}

// ErrNotEnoughData is returned when there is not enough data to
// create the ObjectStorage.
var ErrNotEnoughData = errors.New("no packfile or idx provided")

// New returns a new ObjectStorage for the packfile at path.
//
// If no idx reader is provided, the index will be generated
// by reading the packfile.
func New(packfile io.ReadSeeker, idx io.Reader) (*ObjectStorage, error) {
	index, err := buildIndex(packfile, idx)
	if err != nil {
		return nil, err
	}

	return &ObjectStorage{
		packfile: packfile,
		index:    index,
	}, nil
}

func buildIndex(packfile io.Reader, idx io.Reader) (index.Index, error) {
	if packfile == nil && idx == nil {
		return nil, ErrNotEnoughData
	}

	if idx != nil {
		return index.NewFromIdx(idx)
	}

	return index.NewFromPackfile(packfile)
}

// New returns a new empty object. Unused method.
func (s *ObjectStorage) New() (core.Object, error) {
	return &memory.Object{}, nil
}

// Set adds a new object to the storage.
// This method always returns an error as this particular
// implementation is read only.
func (s *ObjectStorage) Set(core.Object) (core.Hash, error) {
	return core.ZeroHash, fmt.Errorf("set operation is not allowed")
}

// Get returns the object with the given hash, by searching the
// packfile.
func (s *ObjectStorage) Get(h core.Hash) (core.Object, error) {
	offset, err := s.index.Get(h)
	if err != nil {
		return nil, err
	}

	return packfile.ObjectAt(s.packfile, offset, s)
}

// Iter returns an iterator for all the objects in the packfile with the
// given type.
func (s *ObjectStorage) Iter(t core.ObjectType) (core.ObjectIter, error) {
	var objects []core.Object

	for hash := range s.index {
		object, err := s.Get(hash)
		if err != nil {
			return nil, err
		}
		if object.Type() == t {
			objects = append(objects, object)
		}
	}

	return core.NewObjectSliceIter(objects), nil
}

// ByHash returns an already seen object given its hash.
//
// Given the nature of this storage, it also returns objects that
// have not yet been seen.
func (s *ObjectStorage) ByHash(hash core.Hash) (core.Object, error) {
	return s.Get(hash)
}

// ByOffset returns an already seen object given its offset.
//
// Given the nature of this storage, it also returns objects that
// have not yet been seen.
func (s *ObjectStorage) ByOffset(offset int64) (core.Object, error) {
	return packfile.ObjectAt(s.packfile, offset, s)
}
