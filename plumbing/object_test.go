package plumbing

import (
	"io"

	. "gopkg.in/check.v1"
)

type ObjectSuite struct{}

var _ = Suite(&ObjectSuite{})

func (s *ObjectSuite) TestObjectToPack(c *C) {
	obj := &dummyObject{}
	otp := NewObjectToPack(obj)
	c.Assert(obj, Equals, otp.Object)
	c.Assert(obj, Equals, otp.Original)
	c.Assert(otp.Base, IsNil)
	c.Assert(otp.IsDelta(), Equals, false)

	original := &dummyObject{}
	delta := &dummyObject{}
	deltaToPack := NewDeltaObjectToPack(otp, original, delta)
	c.Assert(obj, Equals, deltaToPack.Object)
	c.Assert(original, Equals, deltaToPack.Original)
	c.Assert(otp, Equals, deltaToPack.Base)
	c.Assert(deltaToPack.IsDelta(), Equals, true)
}

func (s *ObjectSuite) TestObjectTypeString(c *C) {
	c.Assert(CommitObject.String(), Equals, "commit")
	c.Assert(TreeObject.String(), Equals, "tree")
	c.Assert(BlobObject.String(), Equals, "blob")
	c.Assert(TagObject.String(), Equals, "tag")
	c.Assert(REFDeltaObject.String(), Equals, "ref-delta")
	c.Assert(OFSDeltaObject.String(), Equals, "ofs-delta")
	c.Assert(AnyObject.String(), Equals, "any")
	c.Assert(ObjectType(42).String(), Equals, "unknown")
}

func (s *ObjectSuite) TestObjectTypeBytes(c *C) {
	c.Assert(CommitObject.Bytes(), DeepEquals, []byte("commit"))
}

func (s *ObjectSuite) TestObjectTypeValid(c *C) {
	c.Assert(CommitObject.Valid(), Equals, true)
	c.Assert(ObjectType(42).Valid(), Equals, false)
}

func (s *ObjectSuite) TestParseObjectType(c *C) {
	for s, e := range map[string]ObjectType{
		"commit":    CommitObject,
		"tree":      TreeObject,
		"blob":      BlobObject,
		"tag":       TagObject,
		"ref-delta": REFDeltaObject,
		"ofs-delta": OFSDeltaObject,
	} {
		t, err := ParseObjectType(s)
		c.Assert(err, IsNil)
		c.Assert(e, Equals, t)
	}

	t, err := ParseObjectType("foo")
	c.Assert(err, Equals, ErrInvalidType)
	c.Assert(t, Equals, InvalidObject)
}

type dummyObject struct{}

func (*dummyObject) Hash() Hash                      { return ZeroHash }
func (*dummyObject) Type() ObjectType                { return InvalidObject }
func (*dummyObject) SetType(ObjectType)              {}
func (*dummyObject) Size() int64                     { return 0 }
func (*dummyObject) SetSize(s int64)                 {}
func (*dummyObject) Reader() (io.ReadCloser, error)  { return nil, nil }
func (*dummyObject) Writer() (io.WriteCloser, error) { return nil, nil }
