package config_test

import (
	"bytes"

	"gopkg.in/src-d/go-git.v4/formats/config"

	. "gopkg.in/check.v1"
)

type DecoderSuite struct{}

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

func (s *DecoderSuite) TestDecodeFailsWithIdentBeforeSection(c *C) {
	t := `
	key=value
	[section]
	key=value
	`
	decodeFails(c, t)
}

func (s *DecoderSuite) TestDecodeFailsWithEmptySectionName(c *C) {
	t := `
	[]
	key=value
	`
	decodeFails(c, t)
}

func (s *DecoderSuite) TestDecodeFailsWithEmptySubsectionName(c *C) {
	t := `
	[remote ""]
	key=value
	`
	decodeFails(c, t)
}

func (s *DecoderSuite) TestDecodeFailsWithBadSubsectionName(c *C) {
	t := `
	[remote origin"]
	key=value
	`
	decodeFails(c, t)
	t = `
	[remote "origin]
	key=value
	`
	decodeFails(c, t)
}

func (s *DecoderSuite) TestDecodeFailsWithTrailingGarbage(c *C) {
	t := `
	[remote]garbage
	key=value
	`
	decodeFails(c, t)
	t = `
	[remote "origin"]garbage
	key=value
	`
	decodeFails(c, t)
}

func (s *DecoderSuite) TestDecodeFailsWithGarbage(c *C) {
	decodeFails(c, "---")
	decodeFails(c, "????")
	decodeFails(c, "[sect\nkey=value")
	decodeFails(c, "sect]\nkey=value")
	decodeFails(c, `[section]key="value`)
	decodeFails(c, `[section]key=value"`)
}

func decodeFails(c *C, text string) {
	r := bytes.NewReader([]byte(text))
	d := config.NewDecoder(r)
	cfg := &config.Config{}
	err := d.Decode(cfg)
	c.Assert(err, NotNil)
}
