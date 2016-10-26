package config_test

import (
	"gopkg.in/src-d/go-git.v4/formats/config"

	. "gopkg.in/check.v1"
)

type SectionSuite struct {
}

var _ = Suite(&SectionSuite{})

func (s *SectionSuite) TestSection_Option(c *C) {
	sect := &config.Section{
		Options: []*config.Option{
			{Key: "key1", Value: "value1"},
			{Key: "key2", Value: "value2"},
			{Key: "key1", Value: "value3"},
		},
	}
	c.Assert(sect.Option("otherkey"), Equals, "")
	c.Assert(sect.Option("key2"), Equals, "value2")
	c.Assert(sect.Option("key1"), Equals, "value3")
}

func (s *SectionSuite) TestSubsection_Option(c *C) {
	sect := &config.Subsection{
		Options: []*config.Option{
			{Key: "key1", Value: "value1"},
			{Key: "key2", Value: "value2"},
			{Key: "key1", Value: "value3"},
		},
	}
	c.Assert(sect.Option("otherkey"), Equals, "")
	c.Assert(sect.Option("key2"), Equals, "value2")
	c.Assert(sect.Option("key1"), Equals, "value3")
}

func (s *SectionSuite) TestSection_RemoveOption(c *C) {
	sect := &config.Section{
		Options: []*config.Option{
			{Key: "key1", Value: "value1"},
			{Key: "key2", Value: "value2"},
			{Key: "key1", Value: "value3"},
		},
	}
	c.Assert(sect.RemoveOption("otherkey"), DeepEquals, sect)

	expected := &config.Section{
		Options: []*config.Option{
			{Key: "key2", Value: "value2"},
		},
	}
	c.Assert(sect.RemoveOption("key1"), DeepEquals, expected)
}

func (s *SectionSuite) TestSubsection_RemoveOption(c *C) {
	sect := &config.Subsection{
		Options: []*config.Option{
			{Key: "key1", Value: "value1"},
			{Key: "key2", Value: "value2"},
			{Key: "key1", Value: "value3"},
		},
	}
	c.Assert(sect.RemoveOption("otherkey"), DeepEquals, sect)

	expected := &config.Subsection{
		Options: []*config.Option{
			{Key: "key2", Value: "value2"},
		},
	}
	c.Assert(sect.RemoveOption("key1"), DeepEquals, expected)
}
