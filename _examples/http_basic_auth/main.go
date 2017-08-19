package main

import (
	"fmt"
	"gopkg.in/src-d/go-git.v4"
	. "gopkg.in/src-d/go-git.v4/_examples"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"gopkg.in/src-d/go-git.v4/_examples/http_basic_auth/server"
	"os"
	"time"
	"github.com/pkg/errors"
)

// Here is an example to configure http client according to our own needs.
func main() {
	CheckArgs("<url>")
	server.DefaultURL = os.Args[1]

	Info("git clone %s", server.DefaultURL)

	r0, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: server.DefaultURL,
	})

	CheckIfError(err)

	Info("git rev-parse HEAD")

	head0, err := r0.Head()
	CheckIfError(err)
	fmt.Println("Original head:", head0.Hash())

	server.WithServer(func(s *server.HTTPBasicAuthServer) {
		<-time.After(2*time.Second)
		// Clone repository using the new client if the protocol is https://
		Info("git clone %s", s.URL)

		r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
			URL: s.URL,
			Auth: githttp.NewBasicAuthMethod(s.Username, s.Password),
		})

		CheckIfError(err)

		// Retrieve the branch pointed by HEAD
		Info("git rev-parse HEAD")

		head, err := r.Head()
		CheckIfError(err)
		fmt.Println("head:", head.Hash())

		if head.Hash() != head0.Hash() {
			CheckIfError(errors.New("Heads not match"))
		}
	})
}
