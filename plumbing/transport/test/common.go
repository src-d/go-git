// Package test implements common test suite for different transport
// implementations.
//
package test

import (
	"io/ioutil"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"

	. "gopkg.in/check.v1"
)

type FetchPackSuite struct {
	Endpoint            transport.Endpoint
	EmptyEndpoint       transport.Endpoint
	NonExistentEndpoint transport.Endpoint
	Client              transport.Client
}

func (s *FetchPackSuite) TestInfoEmpty(c *C) {
	r, err := s.Client.NewFetchPackSession(s.EmptyEndpoint)
	c.Assert(err, IsNil)
	info, err := r.AdvertisedReferences()
	c.Assert(err, Equals, transport.ErrEmptyRemoteRepository)
	c.Assert(info, IsNil)
}

func (s *FetchPackSuite) TestInfoNotExists(c *C) {
	r, err := s.Client.NewFetchPackSession(s.NonExistentEndpoint)
	c.Assert(err, IsNil)
	info, err := r.AdvertisedReferences()
	c.Assert(err, Equals, transport.ErrRepositoryNotFound)
	c.Assert(info, IsNil)

	r, err = s.Client.NewFetchPackSession(s.NonExistentEndpoint)
	c.Assert(err, IsNil)
	req := &transport.UploadPackRequest{}
	req.Want(plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
	reader, err := r.FetchPack(req)
	c.Assert(err, Equals, transport.ErrRepositoryNotFound)
	c.Assert(reader, IsNil)
}

func (s *FetchPackSuite) TestCannotCallAdvertisedReferenceTwice(c *C) {
	r, err := s.Client.NewFetchPackSession(s.Endpoint)
	c.Assert(err, IsNil)
	_, err = r.AdvertisedReferences()
	c.Assert(err, IsNil)
	_, err = r.AdvertisedReferences()
	c.Assert(err, Equals, transport.ErrAdvertistedReferencesAlreadyCalled)
}

func (s *FetchPackSuite) TestDefaultBranch(c *C) {
	r, err := s.Client.NewFetchPackSession(s.Endpoint)
	c.Assert(err, IsNil)
	defer func() { c.Assert(r.Close(), IsNil) }()

	info, err := r.AdvertisedReferences()
	c.Assert(err, IsNil)
	c.Assert(info.Capabilities.SymbolicReference("HEAD"), Equals, "refs/heads/master")
}

func (s *FetchPackSuite) TestCapabilities(c *C) {
	r, err := s.Client.NewFetchPackSession(s.Endpoint)
	c.Assert(err, IsNil)
	defer func() { c.Assert(r.Close(), IsNil) }()

	info, err := r.AdvertisedReferences()
	c.Assert(err, IsNil)
	c.Assert(info.Capabilities.Get("agent").Values, HasLen, 1)
}

func (s *FetchPackSuite) TestFullFetchPack(c *C) {
	r, err := s.Client.NewFetchPackSession(s.Endpoint)
	c.Assert(err, IsNil)
	defer func() { c.Assert(r.Close(), IsNil) }()

	info, err := r.AdvertisedReferences()
	c.Assert(err, IsNil)
	c.Assert(info, NotNil)

	req := &transport.UploadPackRequest{}
	req.Want(plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))

	reader, err := r.FetchPack(req)
	c.Assert(err, IsNil)

	b, err := ioutil.ReadAll(reader)
	c.Assert(err, IsNil)
	c.Assert(b, HasLen, 85374)
}

func (s *FetchPackSuite) TestFetchPack(c *C) {
	r, err := s.Client.NewFetchPackSession(s.Endpoint)
	c.Assert(err, IsNil)
	defer func() { c.Assert(r.Close(), IsNil) }()

	req := &transport.UploadPackRequest{}
	req.Want(plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))

	reader, err := r.FetchPack(req)
	c.Assert(err, IsNil)

	b, err := ioutil.ReadAll(reader)
	c.Assert(err, IsNil)
	c.Assert(b, HasLen, 85374)
}

func (s *FetchPackSuite) TestFetchPackNoChanges(c *C) {
	r, err := s.Client.NewFetchPackSession(s.Endpoint)
	c.Assert(err, IsNil)
	defer func() { c.Assert(r.Close(), IsNil) }()

	req := &transport.UploadPackRequest{}
	req.Want(plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
	req.Have(plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))

	reader, err := r.FetchPack(req)
	c.Assert(err, Equals, transport.ErrEmptyUploadPackRequest)
	c.Assert(reader, IsNil)
}

func (s *FetchPackSuite) TestFetchPackMulti(c *C) {
	r, err := s.Client.NewFetchPackSession(s.Endpoint)
	c.Assert(err, IsNil)
	defer func() { c.Assert(r.Close(), IsNil) }()

	req := &transport.UploadPackRequest{}
	req.Want(plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
	req.Want(plumbing.NewHash("e8d3ffab552895c19b9fcf7aa264d277cde33881"))

	reader, err := r.FetchPack(req)
	c.Assert(err, IsNil)

	b, err := ioutil.ReadAll(reader)
	c.Assert(err, IsNil)
	c.Assert(b, HasLen, 85585)
}

func (s *FetchPackSuite) TestFetchError(c *C) {
	r, err := s.Client.NewFetchPackSession(s.Endpoint)
	c.Assert(err, IsNil)

	req := &transport.UploadPackRequest{}
	req.Want(plumbing.NewHash("1111111111111111111111111111111111111111"))

	reader, err := r.FetchPack(req)
	c.Assert(err, Equals, transport.ErrEmptyUploadPackRequest)
	c.Assert(reader, IsNil)

	//XXX: We do not test Close error, since implementations might return
	//     different errors if a previous error was found.
}
