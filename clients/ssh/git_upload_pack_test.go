package ssh

import (
	"io/ioutil"
	"testing"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v2/clients/common"
	"gopkg.in/src-d/go-git.v2/core"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteRemote struct{}

var _ = Suite(&SuiteRemote{})

const fixtureRepo = "git@github.com:tyba/git-fixture.git"

func (s *SuiteRemote) TestConnect(c *C) {
	r := NewGitUploadPackService()
	c.Assert(r.Connect(fixtureRepo), ErrorMatches, "cannot connect: Auth required")
}

func (s *SuiteRemote) TestConnectWithAuth(c *C) {
	auth := NewSSHAgent("")
	r := NewGitUploadPackService()
	c.Assert(r.ConnectWithAuth(fixtureRepo, auth), IsNil)
	c.Assert(r.auth, Equals, auth)
}

type mockAuth struct{}

func (*mockAuth) Name() string   { return "" }
func (*mockAuth) String() string { return "" }

func (s *SuiteRemote) TestConnectWithAuthWrongType(c *C) {
	r := NewGitUploadPackService()
	c.Assert(r.ConnectWithAuth(fixtureRepo, &mockAuth{}), Equals, InvalidAuthMethodErr)
}

func (s *SuiteRemote) TestDefaultBranch(c *C) {
	r := NewGitUploadPackService()
	auth := NewSSHAgent("")
	c.Assert(r.ConnectWithAuth(fixtureRepo, auth), IsNil)

	info, err := r.Info()
	c.Assert(err, IsNil)
	c.Assert(info.Capabilities.SymbolicReference("HEAD"), Equals, "refs/heads/master")
}

func (s *SuiteRemote) TestCapabilities(c *C) {
	r := NewGitUploadPackService()
	auth := NewSSHAgent("")
	c.Assert(r.ConnectWithAuth(fixtureRepo, auth), IsNil)

	info, err := r.Info()
	c.Assert(err, IsNil)
	c.Assert(info.Capabilities.Get("agent").Values, HasLen, 1)
}

func (s *SuiteRemote) TestFetch(c *C) {
	r := NewGitUploadPackService()
	auth := NewSSHAgent("")
	c.Assert(r.ConnectWithAuth(fixtureRepo, auth), IsNil)

	reader, err := r.Fetch(&common.GitUploadPackRequest{
		Want: []core.Hash{
			core.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"),
		},
	})

	c.Assert(err, IsNil)

	b, err := ioutil.ReadAll(reader)
	c.Assert(err, IsNil)
	c.Assert(b, HasLen, 85374)
}
