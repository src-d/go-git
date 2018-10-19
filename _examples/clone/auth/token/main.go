package main

import (
	"fmt"
	"os"

	git "gopkg.in/src-d/go-git.v4"
	. "gopkg.in/src-d/go-git.v4/_examples"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

func main() {
	CheckArgs("<url>", "<directory>", "<github_access_token>")
	url, directory, token := os.Args[1], os.Args[2], os.Args[3]

	// Clone the given repository to the given directory
	Info("git clone %s %s", url, directory)

	r, err := git.PlainClone(directory, false, &git.CloneOptions{
		// go-git does not yet support TokenAuth (https://godoc.org/gopkg.in/src-d/go-git.v4/plumbing/transport/http#TokenAuth)
		//
		// Auth: &http.TokenAuth{
		// 	Token: token,
		// },
		//
		// you will recieve and error similar to the following:
		// unexpected client error: unexpected requesting ... status code: 400
		//
		// instead, you must use BasicAuth with your GitHub Access Token as the password
		// and the Username can be anything.
		//
		// here is a StackOverflow post: https://stackoverflow.com/questions/47359377/go-git-how-to-authenticate-at-remote-server
		Auth: &http.BasicAuth{
			Username: "abc123", // yes, this can be anything except an empty string
			Password: token,
		},
		URL:      url,
		Progress: os.Stdout,
	})
	CheckIfError(err)

	// ... retrieving the branch being pointed by HEAD
	ref, err := r.Head()
	CheckIfError(err)
	// ... retrieving the commit object
	commit, err := r.CommitObject(ref.Hash())
	CheckIfError(err)

	fmt.Println(commit)
}
