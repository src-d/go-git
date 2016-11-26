package sideband

import (
	"bytes"
	"io/ioutil"
	"testing"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v4/plumbing/format/pktline"
)

func Test(t *testing.T) { TestingT(t) }

type SidebandSuite struct{}

var _ = Suite(&SidebandSuite{})

func (s *SidebandSuite) TestDecode(c *C) {
	expected := []byte("abcdefghijklmnopqrstuvwxyz")

	buf := bytes.NewBuffer(nil)
	e := pktline.NewEncoder(buf)
	e.Encode(append(PackData.Bytes(), expected[0:8]...))
	e.Encode(append(PackData.Bytes(), expected[8:16]...))
	e.Encode(append(PackData.Bytes(), expected[16:26]...))

	content := make([]byte, 26)
	d := NewDemuxer(Sideband64k, buf)
	n, err := d.Read(content)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 26)
	c.Assert(content, DeepEquals, expected)
}

func (s *SidebandSuite) TestDecodeWithError(c *C) {
	expected := []byte("abcdefghijklmnopqrstuvwxyz")

	buf := bytes.NewBuffer(nil)
	e := pktline.NewEncoder(buf)
	e.Encode(append(PackData.Bytes(), expected[0:8]...))
	e.Encode(append(ErrorMessage.Bytes(), 'F', 'O', 'O', '\n'))
	e.Encode(append(PackData.Bytes(), expected[8:16]...))
	e.Encode(append(PackData.Bytes(), expected[16:26]...))

	content := make([]byte, 26)
	d := NewDemuxer(Sideband64k, buf)
	n, err := d.Read(content)
	c.Assert(err, ErrorMatches, "unexepcted error: FOO\n")
	c.Assert(n, Equals, 8)
	c.Assert(content[0:8], DeepEquals, expected[0:8])
}

func (s *SidebandSuite) TestDecodeWithProgress(c *C) {
	expected := []byte("abcdefghijklmnopqrstuvwxyz")

	buf := bytes.NewBuffer(nil)
	e := pktline.NewEncoder(buf)
	e.Encode(append(PackData.Bytes(), expected[0:8]...))
	e.Encode(append(ProgressMessage.Bytes(), 'F', 'O', 'O', '\n'))
	e.Encode(append(PackData.Bytes(), expected[8:16]...))
	e.Encode(append(PackData.Bytes(), expected[16:26]...))

	content := make([]byte, 26)
	d := NewDemuxer(Sideband64k, buf)
	n, err := d.Read(content)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 26)
	c.Assert(content, DeepEquals, expected)

	progress, err := ioutil.ReadAll(d.Progress)
	c.Assert(err, IsNil)
	c.Assert(progress, DeepEquals, []byte{'F', 'O', 'O', '\n'})
}

func (s *SidebandSuite) TestDecodeWithUnknownChannel(c *C) {

	buf := bytes.NewBuffer(nil)
	e := pktline.NewEncoder(buf)
	e.Encode([]byte{'4', 'F', 'O', 'O', '\n'})

	content := make([]byte, 26)
	d := NewDemuxer(Sideband64k, buf)
	n, err := d.Read(content)
	c.Assert(err, ErrorMatches, "unknown channel 4FOO\n")
	c.Assert(n, Equals, 0)
}

func (s *SidebandSuite) TestDecodeWithPending(c *C) {
	expected := []byte("abcdefghijklmnopqrstuvwxyz")

	buf := bytes.NewBuffer(nil)
	e := pktline.NewEncoder(buf)
	e.Encode(append(PackData.Bytes(), expected[0:8]...))
	e.Encode(append(PackData.Bytes(), expected[8:16]...))
	e.Encode(append(PackData.Bytes(), expected[16:26]...))

	content := make([]byte, 13)
	d := NewDemuxer(Sideband64k, buf)
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
	e.Encode(bytes.Repeat(PackData.Bytes(), MaxPackedSize+1))

	content := make([]byte, 13)
	d := NewDemuxer(Sideband, buf)
	n, err := d.Read(content)
	c.Assert(err, Equals, ErrMaxPackedExceeded)
	c.Assert(n, Equals, 0)

}
