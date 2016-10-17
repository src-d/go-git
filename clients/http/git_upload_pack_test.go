package http

import (
	"io/ioutil"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v4/clients/common"
	"gopkg.in/src-d/go-git.v4/core"
)

type RemoteSuite struct {
	Endpoint common.Endpoint
}

var _ = Suite(&RemoteSuite{})

func (s *RemoteSuite) SetUpSuite(c *C) {
	var err error
	s.Endpoint, err = common.NewEndpoint("https://github.com/git-fixtures/basic")
	c.Assert(err, IsNil)
}

func (s *RemoteSuite) TestNewGitUploadPackServiceAuth(c *C) {
	e, err := common.NewEndpoint("https://foo:bar@github.com/git-fixtures/basic")
	c.Assert(err, IsNil)

	r := NewGitUploadPackService(e)
	auth := r.(*GitUploadPackService).auth

	c.Assert(auth.String(), Equals, "http-basic-auth - foo:*******")
}

func (s *RemoteSuite) TestConnect(c *C) {
	r := NewGitUploadPackService(s.Endpoint)
	c.Assert(r.Connect(), IsNil)
}

func (s *RemoteSuite) TestSetAuth(c *C) {
	auth := &BasicAuth{}
	r := NewGitUploadPackService(s.Endpoint)
	r.SetAuth(auth)
	c.Assert(auth, Equals, r.(*GitUploadPackService).auth)
}

type mockAuth struct{}

func (*mockAuth) Name() string   { return "" }
func (*mockAuth) String() string { return "" }

func (s *RemoteSuite) TestSetAuthWrongType(c *C) {
	r := NewGitUploadPackService(s.Endpoint)
	c.Assert(r.SetAuth(&mockAuth{}), Equals, common.ErrInvalidAuthMethod)
}

func (s *RemoteSuite) TestInfoEmpty(c *C) {
	endpoint, _ := common.NewEndpoint("https://github.com/git-fixture/empty")
	r := NewGitUploadPackService(endpoint)
	c.Assert(r.Connect(), IsNil)

	info, err := r.Info()
	c.Assert(err, Equals, common.ErrAuthorizationRequired)
	c.Assert(info, IsNil)
}

func (s *RemoteSuite) TestInfoNotExists(c *C) {
	endpoint, _ := common.NewEndpoint("https://github.com/git-fixture/not-exists")
	r := NewGitUploadPackService(endpoint)
	c.Assert(r.Connect(), IsNil)

	info, err := r.Info()
	c.Assert(err, Equals, common.ErrAuthorizationRequired)
	c.Assert(info, IsNil)
}

func (s *RemoteSuite) TestDefaultBranch(c *C) {
	r := NewGitUploadPackService(s.Endpoint)
	c.Assert(r.Connect(), IsNil)

	info, err := r.Info()
	c.Assert(err, IsNil)
	c.Assert(info.Capabilities.SymbolicReference("HEAD"), Equals, "refs/heads/master")
}

func (s *RemoteSuite) TestCapabilities(c *C) {
	r := NewGitUploadPackService(s.Endpoint)
	c.Assert(r.Connect(), IsNil)

	info, err := r.Info()
	c.Assert(err, IsNil)
	c.Assert(info.Capabilities.Get("agent").Values, HasLen, 1)
}

func (s *RemoteSuite) TestFetch(c *C) {
	r := NewGitUploadPackService(s.Endpoint)
	c.Assert(r.Connect(), IsNil)

	req := &common.GitUploadPackRequest{}
	req.Want(core.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))

	reader, err := r.Fetch(req)
	c.Assert(err, IsNil)

	b, err := ioutil.ReadAll(reader)
	c.Assert(err, IsNil)
	c.Assert(b, HasLen, 85374)
}

func (s *RemoteSuite) TestFetchNoChanges(c *C) {
	r := NewGitUploadPackService(s.Endpoint)
	c.Assert(r.Connect(), IsNil)

	req := &common.GitUploadPackRequest{}
	req.Want(core.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
	req.Have(core.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))

	reader, err := r.Fetch(req)
	c.Assert(err, Equals, common.ErrEmptyGitUploadPack)
	c.Assert(reader, IsNil)
}

func (s *RemoteSuite) TestFetchMulti(c *C) {
	r := NewGitUploadPackService(s.Endpoint)
	c.Assert(r.Connect(), IsNil)

	req := &common.GitUploadPackRequest{}
	req.Want(core.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
	req.Want(core.NewHash("e8d3ffab552895c19b9fcf7aa264d277cde33881"))

	reader, err := r.Fetch(req)
	c.Assert(err, IsNil)

	b, err := ioutil.ReadAll(reader)
	c.Assert(err, IsNil)
	c.Assert(b, HasLen, 85585)
}
