package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/src-d/go-git.v4/fixtures"

	. "gopkg.in/check.v1"
)

type UploadPackSuite struct {
	fixtures.Suite
	Server *Server
	Path   string
	URL    string
	Bin    string
}

var _ = Suite(&UploadPackSuite{})

func (s *UploadPackSuite) SetUpSuite(c *C) {
	s.Suite.SetUpSuite(c)

	if err := exec.Command("git", "--version").Run(); err != nil {
		c.Skip("git command not found")
	}

	s.Server = DefaultServer

	wd, err := os.Getwd()
	c.Assert(err, IsNil)
	s.Bin = filepath.Clean(filepath.Join(wd, "git-upload-pack"))

	binDir := c.MkDir()
	s.Bin = filepath.Join(binDir, "git-upload-pack")
	f, err := os.OpenFile(s.Bin, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
	c.Assert(err, IsNil)
	_, err = fmt.Fprintf(f, `#!/bin/bash
exec go run "%s/../examples/git-upload-pack/main.go" "$@"`, fixtures.RootFolder)
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)

	fixture := fixtures.Basic().One()
	s.Path = fixture.DotGit().Base()
	s.URL = fmt.Sprintf("file://%s", s.Path)
}

func (s *UploadPackSuite) TestClone(c *C) {
	pathToClone := c.MkDir()

	cmd := exec.Command("git", "clone",
		"--upload-pack", s.Bin,
		s.URL, pathToClone,
	)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GIT_TRACE=true", "GIT_TRACE_PACKET=true")
	stdout, stderr, err := execAndGetOutput(c, cmd)
	c.Assert(err, IsNil, Commentf("STDOUT:\n%s\nSTDERR:\n%s\n", stdout, stderr))
}

func execAndGetOutput(c *C, cmd *exec.Cmd) (stdout, stderr string, err error) {
	sout, err := cmd.StdoutPipe()
	c.Assert(err, IsNil)
	serr, err := cmd.StderrPipe()
	c.Assert(err, IsNil)

	outChan := make(chan string, 1)
	outDoneChan := make(chan error, 1)
	errChan := make(chan string, 1)
	errDoneChan := make(chan error, 1)

	c.Assert(cmd.Start(), IsNil)

	go func() {
		b, err := ioutil.ReadAll(sout)
		if err != nil {
			outDoneChan <- err
		}

		outChan <- string(b)
		outDoneChan <- nil
	}()

	go func() {
		b, err := ioutil.ReadAll(serr)
		if err != nil {
			errDoneChan <- err
		}

		errChan <- string(b)
		errDoneChan <- nil
	}()

	if err = cmd.Wait(); err != nil {
		return <-outChan, <-errChan, err
	}

	if err := <-outDoneChan; err != nil {
		return <-outChan, <-errChan, err
	}

	return <-outChan, <-errChan, <-errDoneChan
}
