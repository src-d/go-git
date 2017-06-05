package main

import (
	"os"

	git "gopkg.in/src-d/go-git.v4"
	. "gopkg.in/src-d/go-git.v4/_examples"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

func main() {
	CheckArgs("<url>", "<directory>")
	url := os.Args[1]
	directory := os.Args[2]

	// Clone the given repository to the given directory
	Info("git clone %s %s --recursive", url, directory)

	opts := git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              nil,
	}

	_, err := git.PlainClone(directory, false, &opts)
	if err != nil {
		if err == transport.ErrAuthorizationRequired {
			Warning("Authentication required")
			Warning("Removing previously created directory")
			os.RemoveAll(directory)
			Warning("Trying again with some fake auth information")

			// ask for authentication credentials and try again...
			opts.Auth = http.NewBasicAuth("foo", "bar")
			_, err = git.PlainClone(directory, false, &opts)
			CheckIfError(err)
		} else {
			CheckIfError(err)
		}
	}
}
