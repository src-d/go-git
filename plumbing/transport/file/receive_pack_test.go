package file

import (
	"fmt"
	"os"
	"os/exec"

	"gopkg.in/src-d/go-git.v4/fixtures"

	. "gopkg.in/check.v1"
)

type ReceivePackSuite struct {
	CommonSuite
	Server     *Server
	RemoteName string
	SrcPath    string
	DstPath    string
	DstURL     string
}

var _ = Suite(&ReceivePackSuite{})

func (s *ReceivePackSuite) SetUpSuite(c *C) {
	s.CommonSuite.SetUpSuite(c)

	s.Server = DefaultServer
	s.RemoteName = "test"

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
	// git <2.0 cannot push to an empty repository without a refspec.
	cmd := exec.Command("git", "push",
		"--receive-pack", s.ReceivePackBin,
		s.RemoteName, "refs/heads/*:refs/heads/*",
	)
	cmd.Dir = s.SrcPath
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GIT_TRACE=true", "GIT_TRACE_PACKET=true")
	stdout, stderr, err := execAndGetOutput(c, cmd)
	c.Assert(err, IsNil, Commentf("STDOUT:\n%s\nSTDERR:\n%s\n", stdout, stderr))
}
