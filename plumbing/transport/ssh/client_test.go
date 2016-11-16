package ssh

import (
	"testing"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"

	. "gopkg.in/check.v1"
	"os"
)

func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct {
	Endpoint transport.Endpoint
}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) SetUpSuite(c *C) {
	var err error
	s.Endpoint, err = transport.NewEndpoint("git@github.com:git-fixtures/basic.git")
	c.Assert(err, IsNil)

	if os.Getenv("SSH_AUTH_SOCK") == "" {
		c.Skip("SSH_AUTH_SOCK is not set")
	}
}

// A mock implementation of client.common.AuthMethod
// to test non ssh auth method detection.
type mockAuth struct{}

func (*mockAuth) Name() string   { return "" }
func (*mockAuth) String() string { return "" }

func (s *ClientSuite) TestSetAuthWrongType(c *C) {
	r := NewClient(s.Endpoint)
	c.Assert(r.SetAuth(&mockAuth{}), Equals, ErrInvalidAuthMethod)
}

func (s *ClientSuite) TestAlreadyConnected(c *C) {
	r := NewClient(s.Endpoint)
	c.Assert(r.Connect(), IsNil)
	defer func() {
		c.Assert(r.Disconnect(), IsNil)
	}()

	c.Assert(r.Connect(), Equals, ErrAlreadyConnected)
}

func (s *ClientSuite) TestDisconnect(c *C) {
	r := NewClient(s.Endpoint)
	c.Assert(r.Connect(), IsNil)
	c.Assert(r.Disconnect(), IsNil)
}

func (s *ClientSuite) TestDisconnectedWhenNonConnected(c *C) {
	r := NewClient(s.Endpoint)
	c.Assert(r.Disconnect(), Equals, ErrNotConnected)
}

func (s *ClientSuite) TestAlreadyDisconnected(c *C) {
	r := NewClient(s.Endpoint)
	c.Assert(r.Connect(), IsNil)
	c.Assert(r.Disconnect(), IsNil)
	c.Assert(r.Disconnect(), Equals, ErrNotConnected)
}

func (s *ClientSuite) TestServeralConnections(c *C) {
	r := NewClient(s.Endpoint)
	c.Assert(r.Connect(), IsNil)
	c.Assert(r.Disconnect(), IsNil)

	c.Assert(r.Connect(), IsNil)
	c.Assert(r.Disconnect(), IsNil)

	c.Assert(r.Connect(), IsNil)
	c.Assert(r.Disconnect(), IsNil)
}
