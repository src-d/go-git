package fs

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/packfile"
	"gopkg.in/src-d/go-git.v3/storage/fs/internal/gitdir"
	"gopkg.in/src-d/go-git.v3/storage/fs/internal/index"
)

// ObjectStorage is an implementation of core.ObjectStorage that
// stores data on disk.
type ObjectStorage struct {
	path  string
	index index.Index
}

func NewFromPath(path string) (s *ObjectStorage, err error) {
	dir, err := gitdir.New(path)
	if err != nil {
		return nil, err
	}

	packfile, err := dir.Packfile()
	if err != nil {
		return nil, err
	}

	idxfile, err := dir.Idxfile()
	if err != nil {
		// if there is no idx file, just keep on, we will manage to create one
		// on the fly.
		if err != gitdir.ErrIdxNotFound {
			return nil, err
		}
	}

	return New(packfile, idxfile)

}

// New returns a new ObjectStorage for the packfile at path.
//
// If no idx reader is provided, the index will be generated
// by reading the packfile.
func New(packfilePath, idxPath string) (s *ObjectStorage, err error) {
	packfile, err := os.Open(packfilePath)
	if err != nil {
		return nil, err
	}

	defer func() {
		errClose := packfile.Close()
		if err == nil {
			err = errClose
		}
	}()

	idxfile, err := os.Open(idxPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		errClose := idxfile.Close()
		if err == nil {
			err = errClose
		}
	}()

	index, err := buildIndex(packfile, idxfile)
	if err != nil {
		return nil, err
	}

	return &ObjectStorage{
		path:  packfilePath,
		index: index,
	}, nil
}

func buildIndex(packfile io.Reader, idx io.Reader) (index.Index, error) {
	if idx != nil {
		return index.NewFromIdx(idx)
	}

	return index.NewFromPackfile(packfile)
}

// Set adds a new object to the storage.
// This method always returns an error as this particular
// implementation is read only.
func (s *ObjectStorage) Set(core.Object) (core.Hash, error) {
	return core.ZeroHash, fmt.Errorf("set operation not permitted")
}

// Get returns the object with the given hash, by searching the
// packfile.
func (s *ObjectStorage) Get(h core.Hash) (core.Object, error) {
	offset, err := s.index.Get(h)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}

	defer func() {
		errClose := f.Close()
		if err == nil {
			err = errClose
		}
	}()

	return packfile.ObjectAt(f, offset, s)
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
	f, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}

	defer func() {
		errClose := f.Close()
		if err == nil {
			err = errClose
		}
	}()

	return packfile.ObjectAt(f, offset, s)
}
