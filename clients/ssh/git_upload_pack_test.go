package ssh

import (
	"io/ioutil"
	"testing"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v2/clients/common"
	"gopkg.in/src-d/go-git.v2/clients/http"
	"gopkg.in/src-d/go-git.v2/core"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteRemote struct{}

var _ = Suite(&SuiteRemote{})

const fixtureRepo = "git@github.com:tyba/git-fixture.git"

func (s *SuiteRemote) TestConnect(c *C) {
	r := NewGitUploadPackService()
	c.Assert(r.Connect(fixtureRepo), Equals, AuthRequiredErr)
}

func (s *SuiteRemote) TestConnectWithDefaultSSHAgent(c *C) {
	auth := NewSSHAgent("")
	r := NewGitUploadPackService()
	c.Assert(r.ConnectWithAuth(fixtureRepo, auth), IsNil)
	c.Assert(r.auth, Equals, auth)
}

func (s *SuiteRemote) TestConnectWithSSHAgent(c *C) {
	auth := NewSSHAgent("SSH_AUTH_SOCK")
	r := NewGitUploadPackService()
	c.Assert(r.ConnectWithAuth(fixtureRepo, auth), IsNil)
	c.Assert(r.auth, Equals, auth)
}

type mockAuth struct{}

func (*mockAuth) Name() string   { return "" }
func (*mockAuth) String() string { return "" }

func (s *SuiteRemote) TestConnectWithHTTPAuth(c *C) {
	r := NewGitUploadPackService()
	c.Assert(r.ConnectWithAuth(fixtureRepo, http.NewBasicAuth("foo", "bla")), Equals, InvalidAuthMethodErr)
}

func (s *SuiteRemote) TestConnectWithAuthWrongType(c *C) {
	r := NewGitUploadPackService()
	c.Assert(r.ConnectWithAuth(fixtureRepo, &mockAuth{}), Equals, InvalidAuthMethodErr)
}

func (s *SuiteRemote) TestAlreadyConnected(c *C) {
	r := NewGitUploadPackService()
	c.Assert(r.ConnectWithAuth(fixtureRepo, NewSSHAgent("")), IsNil)
	c.Assert(r.ConnectWithAuth(fixtureRepo, NewSSHAgent("")), Equals, AlreadyConnectedErr)
}

func (s *SuiteRemote) TestDisconnect(c *C) {
	r := NewGitUploadPackService()
	c.Assert(r.ConnectWithAuth(fixtureRepo, NewSSHAgent("")), IsNil)
	c.Assert(r.Disconnect(), IsNil)
}

func (s *SuiteRemote) TestAlreadyDisconnected(c *C) {
	r := NewGitUploadPackService()
	c.Assert(r.ConnectWithAuth(fixtureRepo, NewSSHAgent("")), IsNil)
	c.Assert(r.Disconnect(), IsNil)
	c.Assert(r.Disconnect(), Equals, NotConnectedErr)
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
