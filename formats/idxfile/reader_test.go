package idxfile

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ReaderSuite struct{}

var _ = Suite(&ReaderSuite{})

func (s *ReaderSuite) TestReadPackfile(c *C) {
	f, err := os.Open("fixtures/git-fixture.idx")
	c.Assert(err, IsNil)

	idx := &Idx{}

	r := NewReader(f)
	l, err := r.Read(idx)
	c.Assert(err, IsNil)
	c.Assert(l, Equals, int64(0))

	c.Assert(int(idx.ObjectCount), Equals, 31)
	c.Assert(idx.Objects, HasLen, 31)
	c.Assert(idx.Objects[0].Hash.String(), Equals, "1669dce138d9b841a518c64b10914d88f5e488ea")
	c.Assert(idx.Objects[0].Offset, Equals, uint64(615))

	c.Assert(fmt.Sprintf("%x", idx.IdxChecksum), Equals, "bba9b7a9895724819225a044c857d391bb9d61d9")
	c.Assert(fmt.Sprintf("%x", idx.PackfileChecksum), Equals, "54bb61360ab2dad1a3e344a8cd3f82b848518cba")

	idx.IdxChecksum = [20]byte{}
	b := bytes.NewBuffer(nil)
	w := NewWriter(b)
	size, err := w.Write(idx)
	c.Assert(err, IsNil)
	c.Assert(size, Equals, 1940)

	c.Assert(fmt.Sprintf("%x", idx.IdxChecksum), Equals, "bba9b7a9895724819225a044c857d391bb9d61d9")
}
