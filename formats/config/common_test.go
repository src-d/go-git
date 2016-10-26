package config_test

import (
	"testing"

	"gopkg.in/src-d/go-git.v4/formats/config"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type CommonSuite struct{}

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

func (s *CommonSuite) TestConfig_RemoveSection(c *C) {
	sect := config.New().
		AddOption("section1", "", "key1", "value1").
		AddOption("section2", "", "key1", "value1")
	expected := config.New().
		AddOption("section1", "", "key1", "value1")
	c.Assert(sect.RemoveSection("other"), DeepEquals, sect)
	c.Assert(sect.RemoveSection("section2"), DeepEquals, expected)
}

func (s *CommonSuite) TestConfig_RemoveSubsection(c *C) {
	sect := config.New().
		AddOption("section1", "sub1", "key1", "value1").
		AddOption("section1", "sub2", "key1", "value1")
	expected := config.New().
		AddOption("section1", "sub1", "key1", "value1")
	c.Assert(sect.RemoveSubsection("section1", "other"), DeepEquals, sect)
	c.Assert(sect.RemoveSubsection("other", "other"), DeepEquals, sect)
	c.Assert(sect.RemoveSubsection("section1", "sub2"), DeepEquals, expected)
}
