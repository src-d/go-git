package file

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/src-d/go-git.v4/fixtures"

	. "gopkg.in/check.v1"
)

type ReceivePackSuite struct {
	fixtures.Suite
	Server     *Server
	RemoteName string
	SrcPath    string
	DstPath    string
	DstURL     string
	Bin        string
}

var _ = Suite(&ReceivePackSuite{})

func (s *ReceivePackSuite) SetUpSuite(c *C) {
	s.Suite.SetUpSuite(c)

	if err := exec.Command("git", "--version").Run(); err != nil {
		c.Skip("git command not found")
	}

	s.Server = DefaultServer
	s.RemoteName = "test"

	binDir := c.MkDir()
	s.Bin = filepath.Join(binDir, "git-receive-pack")
	f, err := os.OpenFile(s.Bin, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
	c.Assert(err, IsNil)
	_, err = fmt.Fprintf(f, `#!/bin/bash
exec go run "%s/../examples/git-receive-pack/main.go" "$@"`, fixtures.RootFolder)
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)

	fixture := fixtures.Basic().One()
	s.SrcPath = fixture.DotGit().Base()

	fixture = fixtures.ByTag("empty").One()
	s.DstPath = fixture.DotGit().Base()
	s.DstURL = fmt.Sprintf("file://%s", s.DstPath)

	cmd := exec.Command("git", "remote", "add", s.RemoteName, s.DstURL)
	cmd.Dir = s.SrcPath
	c.Assert(cmd.Run(), IsNil)
}

func (s *ReceivePackSuite) TestPush(c *C) {
	cmd := exec.Command("git", "push",
		"--receive-pack", s.Bin,
		s.RemoteName,
	)
	cmd.Dir = s.SrcPath
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GIT_TRACE=true", "GIT_TRACE_PACKET=true")
	stdout, stderr, err := execAndGetOutput(c, cmd)
	c.Assert(err, IsNil, Commentf("STDOUT:\n%s\nSTDERR:\n%s\n", stdout, stderr))
}
