package idxfile

import (
	"bytes"
	"io"
	"os"

	. "gopkg.in/check.v1"
)

func (s *IdxfileSuite) TestEncode(c *C) {
	for i, path := range [...]string{
		"fixtures/git-fixture.idx",
		"../packfile/fixtures/spinnaker-spinnaker.idx",
	} {
		comment := Commentf("subtest %d: path = %s", i, path)

		expected, idx, err := decode(path)
		c.Assert(err, IsNil, comment)

		obtained := new(bytes.Buffer)
		encoder := NewEncoder(obtained)
		size, err := encoder.Encode(idx)
		c.Assert(err, IsNil, comment)

		c.Assert(size, Equals, expected.Len(), comment)
		c.Assert(obtained, DeepEquals, expected, comment)
	}
}

func decode(path string) (*bytes.Buffer, *Idxfile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	contents := new(bytes.Buffer)
	tee := io.TeeReader(file, contents)

	decoder := NewDecoder(tee)
	idx := &Idxfile{}
	if err = decoder.Decode(idx); err != nil {
		return nil, nil, err
	}

	return contents, idx, file.Close()
}
