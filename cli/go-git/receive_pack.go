package main

import (
	"fmt"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/file"
)

type CmdReceivePack struct {
	cmd

	Args struct {
		GitDir string `positional-arg-name:"git-dir" required:"true"`
	} `positional-args:"yes"`
}

func (CmdReceivePack) Usage() string {
	//XXX: git-receive-pack returns error code 129 if arguments are invalid.
	return fmt.Sprintf("usage: %s <git-dir>", os.Args[0])
}

func (c *CmdReceivePack) Execute(args []string) error {
	gitDir, err := resolvePath(c.Args.GitDir)
	if err != nil {
		return err
	}

	if err := file.DefaultServer.Serve(
		transport.ReceivePackServiceName, gitDir, "",
	); err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err)
		os.Exit(128)
	}

	return nil
}
