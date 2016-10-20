package index

import (
	"testing"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v4/fixtures"
)

func Test(t *testing.T) { TestingT(t) }

type IdxfileSuite struct {
	fixtures.Suite
}

var _ = Suite(&IdxfileSuite{})

func (s *IdxfileSuite) TestDecode(c *C) {
	f, err := fixtures.Basic().One().DotGit().Open("index")
	c.Assert(err, IsNil)

	idx := &Index{}
	d := NewDecoder(f)
	err = d.Decode(idx)
	c.Assert(err, IsNil)

	c.Assert(idx.Version, Equals, uint32(2))
	c.Assert(idx.EntryCount, Equals, uint32(9))
}

func (s *IdxfileSuite) TestDecodeEntries(c *C) {
	f, err := fixtures.Basic().One().DotGit().Open("index")
	c.Assert(err, IsNil)

	idx := &Index{}
	d := NewDecoder(f)
	err = d.Decode(idx)
	c.Assert(err, IsNil)

	c.Assert(idx.Entries, HasLen, 9)

	e := idx.Entries[0]
	c.Assert(e.CreatedAt.Unix(), Equals, int64(1473350251))
	c.Assert(e.CreatedAt.Nanosecond(), Equals, 12059307)
	c.Assert(e.ModifiedAt.Unix(), Equals, int64(1473350251))
	c.Assert(e.ModifiedAt.Nanosecond(), Equals, 12059307)
	c.Assert(e.Dev, Equals, uint32(38))
	c.Assert(e.Inode, Equals, uint32(1715795))
	c.Assert(e.UID, Equals, uint32(1000))
	c.Assert(e.GID, Equals, uint32(100))
	c.Assert(e.Size, Equals, uint32(189))
	c.Assert(e.Hash.String(), Equals, "32858aad3c383ed1ff0a0f9bdf231d54a00c9e88")
	c.Assert(e.Name, Equals, ".gitignore")

	e = idx.Entries[1]
	c.Assert(e.Name, Equals, "CHANGELOG")
}
