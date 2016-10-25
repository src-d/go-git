package config_test

import (
"testing"

. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v4/formats/config"
)

func Test(t *testing.T) { TestingT(t) }

type Fixture struct {
	Text   string
	Config *config.Config
}

var fixtures = map[string]*Fixture{
	"simple": {
		Text: `
		[core]
		repositoryformatversion = 0
		`,
		Config: &config.Config{
			Sections: []*config.Section{
				{
					Name: "core",
					Options: []*config.Option{
						{
							Key: "repositoryformatversion",
							Value: "0",
						},
					},
				},
			},
		},
	},
}
