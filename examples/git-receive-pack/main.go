package main

import (
	"fmt"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/file"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: git receive-pack <git-dir>")
		os.Exit(129)
	}

	if err := file.DefaultServer.Serve(
		transport.ReceivePackServiceName, os.Args[1:],
	); err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err)
		os.Exit(128)
	}
}
