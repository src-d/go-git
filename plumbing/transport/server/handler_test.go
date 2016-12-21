package server

import (
	"testing"

	"gopkg.in/src-d/go-git.v4/fixtures"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
	"gopkg.in/src-d/go-git.v4/storage/memory"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type HandlerSuite struct {
	fixtures.Suite
}

var _ = Suite(&HandlerSuite{})

func (s *HandlerSuite) TestClone(c *C) {
	f := fixtures.Basic().One()
	src, err := filesystem.NewStorage(f.DotGit())
	c.Assert(err, IsNil)

	sess, err := DefaultHandler.NewUploadPackSession(src)
	c.Assert(err, IsNil)

	ar, err := sess.AdvertisedReferences()
	c.Assert(err, IsNil)
	c.Assert(len(ar.References), Equals, 5)

	req := packp.NewUploadPackRequestFromCapabilities(ar.Capabilities)
	req.Wants = append(req.Wants, *ar.Head)

	resp, err := sess.UploadPack(req)
	c.Assert(err, IsNil)
	c.Assert(resp, NotNil)
}

func (s *HandlerSuite) TestPushToEmpty(c *C) {
	dst := memory.NewStorage()
	sess, err := DefaultHandler.NewReceivePackSession(dst)
	c.Assert(err, IsNil)

	ar, err := sess.AdvertisedReferences()
	c.Assert(err, IsNil)
	c.Assert(len(ar.References), Equals, 0)

	req := packp.NewReferenceUpdateRequestFromCapabilities(ar.Capabilities)

	resp, err := sess.ReceivePack(req)
	c.Assert(err, IsNil)
	c.Assert(resp, NotNil)
	c.Assert(resp.Error(), IsNil)
}
