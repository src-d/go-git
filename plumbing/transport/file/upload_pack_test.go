package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"gopkg.in/src-d/go-git.v4/fixtures"

	. "gopkg.in/check.v1"
)

type UploadPackSuite struct {
	CommonSuite
	Server *Server
	Path   string
	URL    string
}

var _ = Suite(&UploadPackSuite{})

func (s *UploadPackSuite) SetUpSuite(c *C) {
	s.CommonSuite.SetUpSuite(c)

	s.Server = DefaultServer

	fixture := fixtures.Basic().One()
	s.Path = fixture.DotGit().Base()
	s.URL = fmt.Sprintf("file://%s", s.Path)
}

func (s *UploadPackSuite) TestClone(c *C) {
	pathToClone := c.MkDir()

	cmd := exec.Command("git", "clone",
		"--upload-pack", s.UploadPackBin,
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
