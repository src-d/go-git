package main

import (
	"fmt"

	"gopkg.in/src-d/go-git.v4"
	. "gopkg.in/src-d/go-git.v4/examples"
)

func main() {
	// We instance an in-memory git repository
	r := git.NewMemoryRepository()

	// Clone the given repository, creating the remote, the local branches
	// and fetching the objects, exactly as:
	Info("git clone https://github.com/src-d/go-siva")

	err := r.Clone(&git.CloneOptions{URL: "https://github.com/src-d/go-siva"})
	CheckIfError(err)

	// Getting the HEAD history from HEAD, just like does:
	Info("git log")

	// ... retrieving the branch being pointed by HEAD
	ref, err := r.Head()
	CheckIfError(err)

	// ... retrieving the commit object
	commit, err := r.Commit(ref.Hash())
	CheckIfError(err)

	// ... we retrieve the commit history
	history, err := commit.History()
	CheckIfError(err)

	// ... now just iterate over the commits, printing it
	for _, c := range history {
		fmt.Print(c)
	}
}
