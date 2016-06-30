package packfile

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v3/core"
)

const (
	sigOffset   = 0
	verOffset   = 4
	countOffset = 8
)

type ParserSuite struct {
	fixtures map[string]*fix
}

type fix struct {
	path     string
	parser   *Parser
	seekable io.Seeker
}

func newFix(path string) (*fix, error) {
	fix := new(fix)
	fix.path = path

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	if err = f.Close(); err != nil {
		return nil, err
	}

	seekable := NewSeekable(bytes.NewReader(data))
	fix.seekable = seekable
	fix.parser = NewParser(seekable)

	return fix, nil
}

func (f *fix) seek(o int64) error {
	_, err := f.seekable.Seek(o, os.SEEK_SET)
	return err
}

var _ = Suite(&ParserSuite{})

func (s *ParserSuite) SetUpSuite(c *C) {
	s.fixtures = make(map[string]*fix)
	for _, fixData := range []struct {
		id   string
		path string
	}{
		{"ofs-deltas", "fixtures/alcortesm-binary-relations.pack"},
		{"ref-deltas", "fixtures/git-fixture.ref-delta"},
	} {
		fix, err := newFix(fixData.path)
		c.Assert(err, IsNil,
			Commentf("setting up fixture id %s: %s", fixData.id, err))

		_, ok := s.fixtures[fixData.id]
		c.Assert(ok, Equals, false,
			Commentf("duplicated fixture id: %s", fixData.id))

		s.fixtures[fixData.id] = fix
	}
}

func (s *ParserSuite) TestSignature(c *C) {
	for id, fix := range s.fixtures {
		com := Commentf("fixture id = %s", id)
		err := fix.seek(sigOffset)
		c.Assert(err, IsNil, com)
		p := fix.parser

		sig, err := p.ReadSignature()
		c.Assert(err, IsNil, com)
		c.Assert(p.IsValidSignature(sig), Equals, true, com)
	}
}

func (s *ParserSuite) TestVersion(c *C) {
	for i, test := range [...]struct {
		fixId    string
		expected uint32
	}{
		{
			fixId:    "ofs-deltas",
			expected: uint32(2),
		}, {
			fixId:    "ref-deltas",
			expected: uint32(2),
		},
	} {
		com := Commentf("test %d) fixture id = %s", i, test.fixId)
		fix, ok := s.fixtures[test.fixId]
		c.Assert(ok, Equals, true, com)

		err := fix.seek(verOffset)
		c.Assert(err, IsNil, com)
		p := fix.parser

		v, err := p.ReadVersion()
		c.Assert(err, IsNil, com)
		c.Assert(v, Equals, test.expected, com)
		c.Assert(p.IsSupportedVersion(v), Equals, true, com)
	}
}

func (s *ParserSuite) TestCount(c *C) {
	for i, test := range [...]struct {
		fixId    string
		expected uint32
	}{
		{
			fixId:    "ofs-deltas",
			expected: uint32(0x50),
		}, {
			fixId:    "ref-deltas",
			expected: uint32(0x1c),
		},
	} {
		com := Commentf("test %d) fixture id = %s", i, test.fixId)
		fix, ok := s.fixtures[test.fixId]
		c.Assert(ok, Equals, true, com)

		err := fix.seek(countOffset)
		c.Assert(err, IsNil, com)
		p := fix.parser

		count, err := p.ReadCount()
		c.Assert(err, IsNil, com)
		c.Assert(count, Equals, test.expected, com)
	}
}

func (s *ParserSuite) TestReadObjectTypeAndLength(c *C) {
	for i, test := range [...]struct {
		fixId     string
		offset    int64
		expType   core.ObjectType
		expLength int64
	}{
		{
			fixId:     "ofs-deltas",
			offset:    12,
			expType:   core.CommitObject,
			expLength: 342,
		}, {
			fixId:     "ofs-deltas",
			offset:    1212,
			expType:   core.OFSDeltaObject,
			expLength: 104,
		}, {
			fixId:     "ofs-deltas",
			offset:    3193,
			expType:   core.TreeObject,
			expLength: 226,
		}, {
			fixId:     "ofs-deltas",
			offset:    3639,
			expType:   core.BlobObject,
			expLength: 90,
		}, {
			fixId:     "ofs-deltas",
			offset:    4504,
			expType:   core.BlobObject,
			expLength: 7107,
		}, {
			fixId:     "ref-deltas",
			offset:    84849,
			expType:   core.REFDeltaObject,
			expLength: 6,
		}, {
			fixId:     "ref-deltas",
			offset:    85070,
			expType:   core.REFDeltaObject,
			expLength: 8,
		},
	} {
		com := Commentf("test %d) fixture id = %s", i, test.fixId)
		fix, ok := s.fixtures[test.fixId]
		c.Assert(ok, Equals, true, com)

		err := fix.seek(test.offset)
		c.Assert(err, IsNil, com)
		p := fix.parser

		typ, length, err := p.ReadObjectTypeAndLength()
		c.Assert(err, IsNil, com)
		c.Assert(typ, Equals, test.expType, com)
		c.Assert(length, Equals, test.expLength, com)
	}
}
