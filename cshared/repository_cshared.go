// +build ignore
package main

import (
	"C"
	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/clients/common"
	. "github.com/src-d/go-git"
)

//export c_Repository
func c_Repository() uint64 {
	repo := &Repository{}
	repo_handle := RegisterObject(repo)
	return uint64(repo_handle)
}

//export c_NewRepository
func c_NewRepository(url string, auth uint64) (uint64, int, string) {
	var repo *Repository
	var err error
	url = CopyString(url)
	if auth != IH {
		real_auth, ok := GetObject(Handle(auth))
		if !ok {
			return IH, ErrorCodeNotFound, MessageNotFound
		}
		repo, err = NewRepository(url, real_auth.(common.AuthMethod))
	} else {
		repo, err = NewRepository(url, nil)
	}
	if err != nil {
		return IH, ErrorCodeInternal, err.Error()
	}
	repo_handle := RegisterObject(repo)
	return uint64(repo_handle), ErrorCodeSuccess, ""
}

//export c_NewPlainRepository
func c_NewPlainRepository() uint64 {
	return uint64(RegisterObject(NewPlainRepository()))
}

//export c_Repository_get_Remotes
func c_Repository_get_Remotes(r uint64) uint64 {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return IH
	}
	repo := obj.(*Repository)
	remotes := repo.Remotes
	remotes_handle := RegisterObject(remotes)
	return uint64(remotes_handle)
}

//export c_Repository_set_Remotes
func c_Repository_set_Remotes(r uint64, val uint64) {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return
	}
	repo := obj.(*Repository)
	obj, ok = GetObject(Handle(val))
	if !ok {
		return
	}
	remotes := obj.(map[string]*Remote)
	repo.Remotes = remotes
}

//export c_Repository_get_Storage
func c_Repository_get_Storage(r uint64) uint64 {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return IH
	}
	repo := obj.(*Repository)
	storage := repo.Storage
	storage_handle := RegisterObject(storage)
	return uint64(storage_handle)
}

//export c_Repository_set_Storage
func c_Repository_set_Storage(r uint64, val uint64) {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return
	}
	repo := obj.(*Repository)
	obj, ok = GetObject(Handle(val))
	if !ok {
		return
	}
	storage := obj.(core.ObjectStorage)
	repo.Storage = storage
}

//export c_Repository_get_URL
func c_Repository_get_URL(r uint64) string {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return ""
	}
	repo := obj.(*Repository)
	return repo.URL
}

//export c_Repository_set_URL
func c_Repository_set_URL(r uint64, val string) {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return
	}
	repo := obj.(*Repository)
	repo.URL = CopyString(val)
}

//export c_Repository_Pull
func c_Repository_Pull(r uint64, remoteName, branch string) (int, string) {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return ErrorCodeNotFound, MessageNotFound
	}
	repo := obj.(*Repository)
	err := repo.Pull(remoteName, CopyString(branch))
	if err == nil {
		return ErrorCodeSuccess, ""
	}
	return ErrorCodeInternal, err.Error()
}

//export c_Repository_PullDefault
func c_Repository_PullDefault(r uint64) (int, string) {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return ErrorCodeNotFound, MessageNotFound
	}
	repo := obj.(*Repository)
	err := repo.PullDefault()
	if err == nil {
		return ErrorCodeSuccess, ""
	}
	return ErrorCodeInternal, err.Error()
}

//export c_Repository_Commit
func c_Repository_Commit(r uint64, h []byte) (uint64, int, string) {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return IH, ErrorCodeNotFound, MessageNotFound
	}
	repo := obj.(*Repository)
	var hash core.Hash
	copy(hash[:], h)
	commit, err := repo.Commit(hash)
	if err != nil {
		return IH, ErrorCodeInternal, err.Error()
	}
  commit_handle := RegisterObject(commit)
	return uint64(commit_handle), ErrorCodeSuccess, ""
}

//export c_Repository_Commits
func c_Repository_Commits(r uint64) uint64 {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return IH
	}
	repo := obj.(*Repository)
	iter := repo.Commits()
	iter_handle := RegisterObject(iter)
	return uint64(iter_handle)
}

//export c_Repository_Tree
func c_Repository_Tree(r uint64, h []byte) (uint64, int, string) {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return IH, ErrorCodeNotFound, MessageNotFound
	}
	repo := obj.(*Repository)
	var hash core.Hash
	copy(hash[:], h)
	tree, err := repo.Tree(hash)
	if err != nil {
		return IH, ErrorCodeInternal, err.Error()
	}
	tree_handle := RegisterObject(tree)
	return uint64(tree_handle), ErrorCodeSuccess, ""
}

//export c_Repository_Blob
func c_Repository_Blob(r uint64, h []byte) (uint64, int, string) {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return IH, ErrorCodeNotFound, MessageNotFound
	}
	repo := obj.(*Repository)
	var hash core.Hash
	copy(hash[:], h)
	blob, err := repo.Blob(hash)
	if err != nil {
		return IH, ErrorCodeInternal, err.Error()
	}
	blob_handle := RegisterObject(blob)
	return uint64(blob_handle), ErrorCodeSuccess, ""
}

//export c_Repository_Tag
func c_Repository_Tag(r uint64, h []byte) (uint64, int, string) {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return IH, ErrorCodeNotFound, MessageNotFound
	}
	repo := obj.(*Repository)
	var hash core.Hash
	copy(hash[:], h)
	tag, err := repo.Tag(hash)
	if err != nil {
		return IH, ErrorCodeInternal, err.Error()
	}
	tag_handle := RegisterObject(tag)
	return uint64(tag_handle), ErrorCodeSuccess, ""
}

//export c_Repository_Tags
func c_Repository_Tags(r uint64) uint64 {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return IH
	}
	repo := obj.(*Repository)
	iter := repo.Tags()
	iter_handle := RegisterObject(iter)
	return uint64(iter_handle)
}

//export c_Repository_Object
func c_Repository_Object(r uint64, h []byte) (uint64, int, string) {
	obj, ok := GetObject(Handle(r))
	if !ok {
		return IH, ErrorCodeNotFound, MessageNotFound
	}
	repo := obj.(*Repository)
	var hash core.Hash
	copy(hash[:], h)
	robj, err := repo.Object(hash)
	if err != nil {
		return IH, ErrorCodeInternal, err.Error()
	}
	robj_handle := RegisterObject(robj)
	return uint64(robj_handle), ErrorCodeSuccess, ""
}
