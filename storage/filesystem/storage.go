// Package filesystem is a storage backend base on filesystems
package filesystem

import (
	"gopkg.in/src-d/go-git.v4/storage/filesystem/internal/dotgit"
	"gopkg.in/src-d/go-git.v4/utils/fs"
)

type Storage struct {
	ObjectStorage
	ReferenceStorage
	ConfigStorage
}

func NewStorage(fs fs.Filesystem) (*Storage, error) {
	dir := dotgit.New(fs)
	o, err := newObjectStorage(dir)
	if err != nil {
		return nil, err
	}

	return &Storage{
		ObjectStorage:    o,
		ReferenceStorage: ReferenceStorage{dir: dir},
		ConfigStorage:    ConfigStorage{dir: dir},
	}, nil
}
