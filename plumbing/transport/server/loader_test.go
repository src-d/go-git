package server

import (
	"os/exec"
	"path/filepath"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"

	. "gopkg.in/check.v1"
)

type LoaderSuite struct {
	RepoPath string
}

var _ = Suite(&LoaderSuite{})

func (s *LoaderSuite) SetUpSuite(c *C) {
	if err := exec.Command("git", "--version").Run(); err != nil {
		c.Skip("git command not found")
	}

	dir := c.MkDir()
	s.RepoPath = filepath.Join(dir, "repo.git")
	c.Assert(exec.Command("git", "init", "--bare", s.RepoPath).Run(), IsNil)
}

func (s *LoaderSuite) TestLoadNonExistent(c *C) {
	sto, err := DefaultLoader.Load("", "/does-not-exist")
	c.Assert(err, Equals, transport.ErrRepositoryNotFound)
	c.Assert(sto, IsNil)
}

func (s *LoaderSuite) TestLoadNonExistentIgnoreHost(c *C) {
	sto, err := DefaultLoader.Load("example.com", "/does-not-exist")
	c.Assert(err, Equals, transport.ErrRepositoryNotFound)
	c.Assert(sto, IsNil)
}

func (s *LoaderSuite) TestLoad(c *C) {
	sto, err := DefaultLoader.Load("", s.RepoPath)
	c.Assert(err, IsNil)
	c.Assert(sto, NotNil)
}

func (s *LoaderSuite) TestLoadIgnoreHost(c *C) {
	sto, err := DefaultLoader.Load("example.com", s.RepoPath)
	c.Assert(err, IsNil)
	c.Assert(sto, NotNil)
}
