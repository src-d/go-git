// Package filesystem is a storage backend base on filesystems
package filesystem

import (
	"gopkg.in/src-d/go-git.v4/plumbing/cache"
	"gopkg.in/src-d/go-git.v4/storage/filesystem/dotgit"

	"gopkg.in/src-d/go-billy.v4"
)

// Storage is an implementation of git.Storer that stores data on disk in the
// standard git format (this is, the .git directory). Zero values of this type
// are not safe to use, see the NewStorage function below.
type Storage struct {
	fs       billy.Filesystem
	commonfs billy.Filesystem
	dir      *dotgit.DotGit

	ObjectStorage
	ReferenceStorage
	IndexStorage
	ShallowStorage
	ConfigStorage
	ModuleStorage
}

// Options holds configuration for the storage.
type Options struct {
	// ExclusiveAccess means that the filesystem is not modified externally
	// while the repo is open.
	ExclusiveAccess bool
	// KeepDescriptors makes the file descriptors to be reused but they will
	// need to be manually closed calling Close().
	KeepDescriptors bool
	// CommonDir sets the directory used for accessing non-worktree files that
	// would normally be taken from the root directory.
	CommonDir billy.Filesystem
}

// NewStorage returns a new Storage backed by a given `fs.Filesystem` and cache.
func NewStorage(fs billy.Filesystem, cache cache.Object) *Storage {
	return NewStorageWithOptions(fs, cache, Options{})
}

// NewStorageWithOptions returns a new Storage with extra options,
// backed by a given `fs.Filesystem` and cache.
func NewStorageWithOptions(fs billy.Filesystem, cache cache.Object, ops Options) *Storage {
	dirOps := dotgit.Options{
		ExclusiveAccess: ops.ExclusiveAccess,
		KeepDescriptors: ops.KeepDescriptors,
		CommonDir:       ops.CommonDir,
	}

	dir := dotgit.NewWithOptions(fs, dirOps)
	if ops.CommonDir == nil {
		ops.CommonDir = fs
	}

	return &Storage{
		fs:       fs,
		commonfs: ops.CommonDir,
		dir:      dir,

		ObjectStorage:    *NewObjectStorageWithOptions(dir, cache, ops),
		ReferenceStorage: ReferenceStorage{dir: dir},
		IndexStorage:     IndexStorage{dir: dir},
		ShallowStorage:   ShallowStorage{dir: dir},
		ConfigStorage:    ConfigStorage{dir: dir},
		ModuleStorage:    ModuleStorage{dir: dir},
	}
}

// Filesystem returns the underlying filesystem
func (s *Storage) Filesystem() billy.Filesystem {
	return s.fs
}

// MainFilesystem returns the underlying filesystem for the main
// working-tree/common git directory
func (s *Storage) MainFilesystem() billy.Filesystem {
	return s.commonfs
}

// Init initializes .git directory
func (s *Storage) Init() error {
	return s.dir.Initialize()
}
