package file

import (
	"os/exec"

	"gopkg.in/src-d/go-git.v4/fixtures"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/test"

	. "gopkg.in/check.v1"
)

type SendPackSuite struct {
	fixtures.Suite
	test.SendPackSuite
}

var _ = Suite(&SendPackSuite{})

func (s *SendPackSuite) SetUpSuite(c *C) {
	s.Suite.SetUpSuite(c)

	if err := exec.Command("git", "--version").Run(); err != nil {
		c.Skip("git command not found")
	}

	s.SendPackSuite.Client = DefaultClient
}

func (s *SendPackSuite) SetUpTest(c *C) {
	fixture := fixtures.Basic().One()
	path := fixture.DotGit().Base()
	s.Endpoint = prepareRepo(c, path)

	fixture = fixtures.ByTag("empty").One()
	path = fixture.DotGit().Base()
	s.EmptyEndpoint = prepareRepo(c, path)

	s.NonExistentEndpoint = prepareRepo(c, "/non-existent")
}

func (s *SendPackSuite) TearDownTest(c *C) {
	s.Suite.TearDownSuite(c)
}
