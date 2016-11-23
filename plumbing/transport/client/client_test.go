package client

import (
	"fmt"
	"testing"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) TestNewClientHTTP(c *C) {
	e, err := transport.NewEndpoint("http://github.com/src-d/go-git")
	c.Assert(err, IsNil)

	output, err := NewClient(e)
	c.Assert(err, IsNil)
	c.Assert(typeAsString(output), Equals, "*http.Client")

	e, err = transport.NewEndpoint("https://github.com/src-d/go-git")
	c.Assert(err, IsNil)

	output, err = NewClient(e)
	c.Assert(err, IsNil)
	c.Assert(typeAsString(output), Equals, "*http.Client")
}

func (s *ClientSuite) TestNewClientSSH(c *C) {
	e, err := transport.NewEndpoint("ssh://github.com/src-d/go-git")
	c.Assert(err, IsNil)

	output, err := NewClient(e)
	c.Assert(err, IsNil)
	c.Assert(typeAsString(output), Equals, "*ssh.Client")
}

func (s *ClientSuite) TestNewClientUnknown(c *C) {
	e, err := transport.NewEndpoint("unknown://github.com/src-d/go-git")
	c.Assert(err, IsNil)

	_, err = NewClient(e)
	c.Assert(err, NotNil)
}

func (s *ClientSuite) TestInstallProtocol(c *C) {
	InstallProtocol("newscheme", newDummyClient)
	c.Assert(Protocols["newscheme"], NotNil)
}

type dummyClient struct {
	*http.Client
}

func newDummyClient(ep transport.Endpoint) transport.Client {
	return &dummyClient{}
}

func typeAsString(v interface{}) string {
	return fmt.Sprintf("%T", v)
}
