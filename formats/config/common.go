// Package config implements decoding and encoding of git config files.
// Reference: https://git-scm.com/docs/git-config
package config

type Config struct {
	Comment  *Comment
	Sections []*Section
	Includes []*Include
}

type Section struct {
	Name        string
	Options     []*Option
	Subsections []*Subsection
}

type Subsection struct {
	Name    string
	Options []*Option
}

type Option struct {
	Key   string
	Value string
}

type Include struct {
	Path   string
	Config *Config
}

type Comment string
