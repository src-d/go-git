package main

import (
	"fmt"
	"os"

	"gopkg.in/src-d/go-git.v4"
)

func main() {
	url := os.Args[1]

	fmt.Printf("Retrieving latest commit from: %q ...\n", url)
	r := git.NewMemoryRepository()

	if err := r.Clone(&git.CloneOptions{URL: url}); err != nil {
		panic(err)
	}

	head, err := r.Head()
	if err != nil {
		panic(err)
	}

	commit, err := r.Commit(head.Hash())
	if err != nil {
		panic(err)
	}

	fmt.Println(commit)
}
