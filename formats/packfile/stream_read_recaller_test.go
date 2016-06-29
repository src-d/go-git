package packfile

import (
	"bytes"
	"fmt"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/storage/memory"

	. "gopkg.in/check.v1"
)

type StreamReadRecallerSuite struct{}

var _ = Suite(&StreamReadRecallerSuite{})

func (s *StreamReadRecallerSuite) TestRead(c *C) {
	data := []byte{0, 1, 2, 3, 4, 5, 7, 8, 9, 10}
	sr := initStreamReader(data)
	all := make([]byte, 0, len(data))

	for len(all) < len(data) {
		tmp := make([]byte, 3)
		nr, err := sr.Read(tmp)
		c.Assert(err, IsNil)
		all = append(all, tmp[:nr]...)
	}
	c.Assert(data, DeepEquals, all)
}

func initStreamReader(data []byte) *StreamReadRecaller {
	buf := bytes.NewBuffer(data)
	return NewStreamReadRecaller(buf)
}

func (s *StreamReadRecallerSuite) TestReadbyte(c *C) {
	data := []byte{0, 1, 2, 3, 4, 5, 7, 8, 9, 10}
	sr := initStreamReader(data)
	all := make([]byte, 0, len(data))

	for len(all) < len(data) {
		b, err := sr.ReadByte()
		c.Assert(err, IsNil)
		all = append(all, b)
	}
	c.Assert(data, DeepEquals, all)
}

func (s *StreamReadRecallerSuite) TestOffsetWithRead(c *C) {
	data := []byte{0, 1, 2, 3, 4, 5, 7, 8, 9, 10}
	sr := initStreamReader(data)
	all := make([]byte, 0, len(data))

	for len(all) < len(data) {
		tmp := make([]byte, 3)
		nr, err := sr.Read(tmp)
		c.Assert(err, IsNil)
		all = append(all, tmp[:nr]...)

		off, err := sr.Offset()
		c.Assert(err, IsNil)
		c.Assert(off, Equals, int64(len(all)))
	}
}

func (s *StreamReadRecallerSuite) TestOffsetWithReadByte(c *C) {
	data := []byte{0, 1, 2, 3, 4, 5, 7, 8, 9, 10}
	sr := initStreamReader(data)
	all := make([]byte, 0, len(data))

	for len(all) < len(data) {
		b, err := sr.ReadByte()
		c.Assert(err, IsNil)
		all = append(all, b)

		off, err := sr.Offset()
		c.Assert(err, IsNil)
		c.Assert(off, Equals, int64(len(all)))
	}
}

func (s *StreamReadRecallerSuite) TestRememberRecall(c *C) {
	sr := NewStreamReadRecaller(bytes.NewBuffer([]byte{}))

	for i, test := range [...]struct {
		off int64
		obj core.Object
		err string // error regexp
	}{
		{off: 0, obj: newObj(0, []byte{'a'})},
		{off: 10, obj: newObj(0, []byte{'b'})},
		{off: 20, obj: newObj(0, []byte{'c'})},
		{off: 30, obj: newObj(0, []byte{'a'}),
			err: "duplicated object: with hash .*"},
		{off: 0, obj: newObj(0, []byte{'d'}),
			err: "duplicated object: with offset 0"},
	} {
		com := Commentf("subtest %d) offset = %d", i, test.off)

		err := sr.Remember(test.off, test.obj)
		if test.err != "" {
			c.Assert(err, ErrorMatches, test.err)
			continue
		}
		c.Assert(err, IsNil, com)

		result, err := sr.RecallByHash(test.obj.Hash())
		c.Assert(err, IsNil)
		c.Assert(result, Equals, test.obj, com)

		result, err = sr.RecallByOffset(test.off)
		c.Assert(err, IsNil)
		c.Assert(result, Equals, test.obj, com)
	}
}

func newObj(typ int, cont []byte) core.Object {
	return memory.NewObject(core.ObjectType(typ), int64(len(cont)), cont)
}

func (s *StreamReadRecallerSuite) TestRecallErrors(c *C) {
	sr := NewStreamReadRecaller(bytes.NewBuffer([]byte{}))
	obj := newObj(0, []byte{})

	_, err := sr.RecallByHash(obj.Hash())
	c.Assert(err, ErrorMatches, ErrCannotRecall.Error()+".*")

	_, err = sr.RecallByOffset(0)
	c.Assert(err, ErrorMatches, ErrCannotRecall.Error()+".*")

	rememberSomeObjects(sr)

	_, err = sr.RecallByHash(obj.Hash())
	c.Assert(err, ErrorMatches, ErrCannotRecall.Error()+".*")

	_, err = sr.RecallByOffset(15)
	c.Assert(err, ErrorMatches, ErrCannotRecall.Error()+".*")

}

func rememberSomeObjects(sr *StreamReadRecaller) error {
	for i, init := range [...]struct {
		off int64
		obj core.Object
	}{
		{off: 0, obj: newObj(0, []byte{'a'})},
		{off: 10, obj: newObj(0, []byte{'b'})},
		{off: 20, obj: newObj(0, []byte{'c'})},
	} {
		err := sr.Remember(init.off, init.obj)
		if err != nil {
			return fmt.Errorf("cannot ask StreamReader to Remember item %d", i)
		}
	}

	return nil
}
