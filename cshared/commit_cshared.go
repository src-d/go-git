package cshared

import (
	"C"
	. "github.com/src-d/go-git"
)

/*
type Commit struct {
	Hash      core.Hash
	Author    Signature
	Committer Signature
	Message   string

	tree    core.Hash
	parents []core.Hash
	r       *Repository
}
 */

//export c_Commit_get_Hash
func c_Commit_get_Hash(c uint64) []byte {
	obj, ok := GetObject(Handle(c))
	if !ok {
		return nil
	}
	commit := obj.(Commit)
	return commit.Hash[:]
}

//export c_Commit_get_Author
func c_Commit_get_Author(c uint64) uint64 {
	obj, ok := GetObject(Handle(c))
	if !ok {
		return IH
	}
	commit := obj.(Commit)
	author := commit.Author
	author_handle := RegisterObject(author)
	return uint64(author_handle)
}

//export c_Commit_get_Committer
func c_Commit_get_Committer(c uint64) uint64 {
	obj, ok := GetObject(Handle(c))
	if !ok {
		return IH
	}
	commit := obj.(Commit)
	committer := commit.Committer
	committer_handle := RegisterObject(committer)
	return uint64(committer_handle)
}

//export c_Commit_get_Message
func c_Commit_get_Message(c uint64) string {
	obj, ok := GetObject(Handle(c))
	if !ok {
		return ""
	}
	commit := obj.(Commit)
	return commit.Message
}
