package ssh

import (
	"io/ioutil"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"

	. "gopkg.in/check.v1"
)

type FetchPackSuite struct {
	Endpoint transport.Endpoint
}

var _ = Suite(&FetchPackSuite{})

func (s *FetchPackSuite) SetUpSuite(c *C) {
	var err error
	s.Endpoint, err = transport.NewEndpoint("git@github.com:git-fixtures/basic.git")
	c.Assert(err, IsNil)

	if os.Getenv("SSH_AUTH_SOCK") == "" {
		c.Skip("SSH_AUTH_SOCK is not set")
	}
}

func (s *FetchPackSuite) TestInfoNotConnected(c *C) {
	r := NewClient(s.Endpoint)
	_, err := r.FetchPackInfo()
	c.Assert(err, Equals, ErrNotConnected)
}

func (s *FetchPackSuite) TestDefaultBranch(c *C) {
	r := NewClient(s.Endpoint)
	c.Assert(r.Connect(), IsNil)
	defer func() { c.Assert(r.Disconnect(), IsNil) }()

	info, err := r.FetchPackInfo()
	c.Assert(err, IsNil)
	c.Assert(info.Capabilities.SymbolicReference("HEAD"), Equals, "refs/heads/master")
}

func (s *FetchPackSuite) TestCapabilities(c *C) {
	r := NewClient(s.Endpoint)
	c.Assert(r.Connect(), IsNil)
	defer func() { c.Assert(r.Disconnect(), IsNil) }()

	info, err := r.FetchPackInfo()
	c.Assert(err, IsNil)
	c.Assert(info.Capabilities.Get("agent").Values, HasLen, 1)
}

func (s *FetchPackSuite) TestFetchNotConnected(c *C) {
	r := NewClient(s.Endpoint)
	pr := &transport.UploadPackRequest{}
	pr.Want(plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
	_, err := r.FetchPack(pr)
	c.Assert(err, Equals, ErrNotConnected)
}

func (s *FetchPackSuite) TestFetch(c *C) {
	r := NewClient(s.Endpoint)
	c.Assert(r.Connect(), IsNil)
	defer func() { c.Assert(r.Disconnect(), IsNil) }()

	req := &transport.UploadPackRequest{}
	req.Want(plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
	req.Want(plumbing.NewHash("e8d3ffab552895c19b9fcf7aa264d277cde33881"))
	reader, err := r.FetchPack(req)
	c.Assert(err, IsNil)
	defer func() { c.Assert(reader.Close(), IsNil) }()

	b, err := ioutil.ReadAll(reader)
	c.Assert(err, IsNil)
	c.Check(len(b), Equals, 85585)
}

func (s *FetchPackSuite) TestFetchError(c *C) {
	r := NewClient(s.Endpoint)
	c.Assert(r.Connect(), IsNil)
	defer func() { c.Assert(r.Disconnect(), IsNil) }()

	req := &transport.UploadPackRequest{}
	req.Want(plumbing.NewHash("1111111111111111111111111111111111111111"))

	reader, err := r.FetchPack(req)
	c.Assert(err, IsNil)

	err = reader.Close()
	c.Assert(err, Not(IsNil))
}
