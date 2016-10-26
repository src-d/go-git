package config_test

import (
	"gopkg.in/src-d/go-git.v4/formats/config"

	. "gopkg.in/check.v1"
)

type OptionSuite struct {
}

var _ = Suite(&OptionSuite{})

func (s *OptionSuite) TestOptions_GetAll(c *C) {
	o := config.Options{
		&config.Option{"k", "v"},
		&config.Option{"ok", "v1"},
		&config.Option{"K", "v2"},
	}
	c.Assert(o.GetAll("k"), DeepEquals, []string{"v", "v2"})
	c.Assert(o.GetAll("K"), DeepEquals, []string{"v", "v2"})
	c.Assert(o.GetAll("ok"), DeepEquals, []string{"v1"})
	c.Assert(o.GetAll("unexistant"), DeepEquals, []string{})

	o = config.Options{}
	c.Assert(o.GetAll("k"), DeepEquals, []string{})
}

func (s *OptionSuite) TestOption_IsKey(c *C) {
	c.Assert((&config.Option{Key: "key"}).IsKey("key"), Equals, true)
	c.Assert((&config.Option{Key: "key"}).IsKey("KEY"), Equals, true)
	c.Assert((&config.Option{Key: "KEY"}).IsKey("key"), Equals, true)
	c.Assert((&config.Option{Key: "key"}).IsKey("other"), Equals, false)
	c.Assert((&config.Option{Key: "key"}).IsKey(""), Equals, false)
	c.Assert((&config.Option{Key: ""}).IsKey("key"), Equals, false)
}
