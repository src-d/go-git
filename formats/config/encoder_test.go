package config_test

import (
	"bytes"

	"gopkg.in/src-d/go-git.v4/formats/config"

	. "gopkg.in/check.v1"
)

type EncoderSuite struct{}

var _ = Suite(&EncoderSuite{})

func (s *EncoderSuite) TestEncode(c *C) {
	for idx, fixture := range fixtures {
		buf := &bytes.Buffer{}
		e := config.NewEncoder(buf)
		err := e.Encode(fixture.Config)
		c.Assert(err, IsNil, Commentf("encoder error for fixture: %d", idx))
		c.Assert(buf.String(), Equals, fixture.Text, Commentf("bad result for fixture: %d", idx))
	}
}
