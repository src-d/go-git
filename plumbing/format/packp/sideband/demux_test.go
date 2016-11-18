package sideband

import (
	"io/ioutil"
	"testing"

	"bytes"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packp/pktline"
)

func Test(t *testing.T) { TestingT(t) }

type SidebandSuite struct{}

var _ = Suite(&SidebandSuite{})

func (s *SidebandSuite) TestDecode(c *C) {
	expected := []byte("abcdefghijklmnopqrstuvwxyz")

	buf := bytes.NewBuffer(nil)
	e := pktline.NewEncoder(buf)
	e.Encode(append([]byte{byte(PackData)}, expected[0:8]...))
	e.Encode(append([]byte{byte(PackData)}, expected[8:16]...))
	e.Encode(append([]byte{byte(PackData)}, expected[16:26]...))

	content := make([]byte, 26)
	d := NewDemuxer(Sideband64k, ioutil.NopCloser(buf))
	n, err := d.Read(content)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 26)
	c.Assert(content, DeepEquals, expected)
}

func (s *SidebandSuite) TestDecodeWithProgress(c *C) {
	expected := []byte("abcdefghijklmnopqrstuvwxyz")

	buf := bytes.NewBuffer(nil)
	e := pktline.NewEncoder(buf)
	e.Encode(append([]byte{byte(PackData)}, expected[0:8]...))
	e.Encode([]byte{byte(ProgressMessage), 'F', 'O', 'O', '\n'})
	e.Encode(append([]byte{byte(PackData)}, expected[8:16]...))
	e.Encode(append([]byte{byte(PackData)}, expected[16:26]...))

	content := make([]byte, 26)
	d := NewDemuxer(Sideband64k, ioutil.NopCloser(buf))
	n, err := d.Read(content)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 26)
	c.Assert(content, DeepEquals, expected)
}

func (s *SidebandSuite) TestDecodeWithPending(c *C) {
	expected := []byte("abcdefghijklmnopqrstuvwxyz")

	buf := bytes.NewBuffer(nil)
	e := pktline.NewEncoder(buf)
	e.Encode(append([]byte{byte(PackData)}, expected[0:8]...))
	e.Encode(append([]byte{byte(PackData)}, expected[8:16]...))
	e.Encode(append([]byte{byte(PackData)}, expected[16:26]...))

	content := make([]byte, 13)
	d := NewDemuxer(Sideband64k, ioutil.NopCloser(buf))
	n, err := d.Read(content)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 13)
	c.Assert(content, DeepEquals, expected[0:13])

	n, err = d.Read(content)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 13)
	c.Assert(content, DeepEquals, expected[13:26])
}

func (s *SidebandSuite) TestDecodeErrMaxPacked(c *C) {
	buf := bytes.NewBuffer(nil)
	e := pktline.NewEncoder(buf)
	e.Encode(bytes.Repeat([]byte{byte(PackData)}, MaxPackedSize+1))

	content := make([]byte, 13)
	d := NewDemuxer(Sideband, ioutil.NopCloser(buf))
	n, err := d.Read(content)
	c.Assert(err, Equals, ErrMaxPackedExceeded)
	c.Assert(n, Equals, 0)

}
