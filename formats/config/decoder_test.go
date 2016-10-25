package config_test

import (
	"bytes"

	"gopkg.in/src-d/go-git.v4/formats/config"

	. "gopkg.in/check.v1"
)

type DecoderSuite struct {
}

var _ = Suite(&DecoderSuite{})

func (s *DecoderSuite) TestDecode(c *C) {
	for idx, fixture := range fixtures {
		r := bytes.NewReader([]byte(fixture.Raw))
		d := config.NewDecoder(r)
		cfg := &config.Config{}
		err := d.Decode(cfg)
		c.Assert(err, IsNil, Commentf("decoder error for fixture: %d", idx))
		c.Assert(cfg, DeepEquals, fixture.Config, Commentf("bad result for fixture: %d", idx))
	}
}
