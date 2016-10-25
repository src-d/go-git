package config_test

import (
	"testing"

	"gopkg.in/src-d/go-git.v4/formats/config"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type CommonSuite struct {
}

var _ = Suite(&CommonSuite{})

func (s *CommonSuite) TestConfig_SetOption(c *C) {
	obtained := config.New().SetOption("section", "", "key1", "value1")
	expected := &config.Config{
		Sections: []*config.Section{
			{
				Name: "section",
				Options: []*config.Option{
					{Key: "key1", Value: "value1"},
				},
			},
		},
	}
	c.Assert(obtained, DeepEquals, expected)
	obtained = obtained.SetOption("section", "", "key1", "value1")
	c.Assert(obtained, DeepEquals, expected)

	obtained = config.New().SetOption("section", "subsection", "key1", "value1")
	expected = &config.Config{
		Sections: []*config.Section{
			{
				Name: "section",
				Subsections: []*config.Subsection{
					{
						Name: "subsection",
						Options: []*config.Option{
							{Key: "key1", Value: "value1"},
						},
					},
				},
			},
		},
	}
	c.Assert(obtained, DeepEquals, expected)
	obtained = obtained.SetOption("section", "subsection", "key1", "value1")
	c.Assert(obtained, DeepEquals, expected)
}

func (s *CommonSuite) TestConfig_AddOption(c *C) {
	obtained := config.New().AddOption("section", "", "key1", "value1")
	expected := &config.Config{
		Sections: []*config.Section{
			{
				Name: "section",
				Options: []*config.Option{
					{Key: "key1", Value: "value1"},
				},
			},
		},
	}
	c.Assert(obtained, DeepEquals, expected)
}

func (s *CommonSuite) TestSection_Option(c *C) {
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

func (s *CommonSuite) TestSubsection_Option(c *C) {
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

func (s *CommonSuite) TestOption_IsKey(c *C) {
	c.Assert((&config.Option{Key: "key"}).IsKey("key"), Equals, true)
	c.Assert((&config.Option{Key: "key"}).IsKey("KEY"), Equals, true)
	c.Assert((&config.Option{Key: "KEY"}).IsKey("key"), Equals, true)
	c.Assert((&config.Option{Key: "key"}).IsKey("other"), Equals, false)
	c.Assert((&config.Option{Key: "key"}).IsKey(""), Equals, false)
	c.Assert((&config.Option{Key: ""}).IsKey("key"), Equals, false)
}