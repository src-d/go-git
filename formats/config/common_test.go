package config_test

import (
	"testing"

	"gopkg.in/src-d/go-git.v4/formats/config"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Fixture struct {
	Text   string
	Config *config.Config
}

var fixtures = []*Fixture{
	{
		Text:   "",
		Config: config.New(),
	},
	{
		Text:   ";Comments only",
		Config: config.New(),
	},
	{
		Text:   "#Comments only",
		Config: config.New(),
	},
	{
		Text:   "[core]\nrepositoryformatversion = 0",
		Config: config.New().AddOption("core", "", "repositoryformatversion", "0"),
	},
	{
		Text:   "[core]\nrepositoryformatversion = 0\n",
		Config: config.New().AddOption("core", "", "repositoryformatversion", "0"),
	},
	{
		Text:   ";Commment\n[core]\n;Comment\nrepositoryformatversion = 0\n",
		Config: config.New().AddOption("core", "", "repositoryformatversion", "0"),
	},
	{
		Text:   "#Commment\n#Comment\n[core]\n#Comment\nrepositoryformatversion = 0\n",
		Config: config.New().AddOption("core", "", "repositoryformatversion", "0"),
	},
	{
		Text:   `
			[sect1]
			opt1 = value1
			[sect1 "subsect1"]
			opt2 = value2
		`,
		Config: config.New().
			AddOption("sect1", "", "opt1", "value1").
			AddOption("sect1", "subsect1", "opt2", "value2"),
	},
}
