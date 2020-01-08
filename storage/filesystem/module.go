package filesystem

import (
	"github.com/goabstract/go-git/plumbing/cache"
	"github.com/goabstract/go-git/storage"
	"github.com/goabstract/go-git/storage/filesystem/dotgit"
)

type ModuleStorage struct {
	dir *dotgit.DotGit
}

func (s *ModuleStorage) Module(name string) (storage.Storer, error) {
	fs, err := s.dir.Module(name)
	if err != nil {
		return nil, err
	}

	return NewStorage(fs, cache.NewObjectLRUDefault()), nil
}
