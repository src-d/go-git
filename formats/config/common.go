// Package config implements decoding, encoding and
// manipulation git config files.
//
// Reference: https://git-scm.com/docs/git-config
package config

import "strings"

// New creates a new config instance.
func New() *Config {
	return &Config{}
}

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
	// Key preserving original caseness.
	// Use IsKey instead to compare key regardless of caseness.
	Key   string
	// Original value as string, could be not notmalized.
	Value string
}

// A reference to a included configuration.
type Include struct {
	Path   string
	Config *Config
}

type Comment string

// IsKey returns true if the given key matches
// this options' key in a case-insensitive comparison.
func (o *Option) IsKey(key string) bool {
	return strings.ToLower(o.Key) == strings.ToLower(key)
}

func (c *Config) Section(name string) *Section {
	for i := len(c.Sections) - 1; i >= 0; i-- {
		s := c.Sections[i]
		if s.Name == name {
			return s
		}
	}
	s := &Section{Name: name}
	c.Sections = append(c.Sections, s)
	return s
}

func (s *Section) Subsection(name string) *Subsection {
	for i := len(s.Subsections) - 1; i >= 0; i-- {
		ss := s.Subsections[i]
		if ss.Name == name {
			return ss
		}
	}
	ss := &Subsection{Name: name}
	s.Subsections = append(s.Subsections, ss)
	return ss
}

func (s *Section) Option(key string) string {
	return getOption(s.Options, key)
}

func (s *Subsection) Option(key string) string {
	return getOption(s.Options, key)
}

func getOption(opts []*Option, key string) string {
	for i := len(opts) - 1; i >= 0; i-- {
		o := opts[i]
		if o.IsKey(key) {
			return o.Value
		}
	}
	return ""
}

func (s *Config) AddOption(section string, subsection string, key string, value string) *Config {
	if subsection == "" {
		s.Section(section).AddOption(key, value)
	} else {
		s.Section(section).Subsection(subsection).AddOption(key, value)
	}

	return s
}

func (s *Config) SetOption(section string, subsection string, key string, value string) *Config {
	if subsection == "" {
		s.Section(section).SetOption(key, value)
	} else {
		s.Section(section).Subsection(subsection).SetOption(key, value)
	}

	return s
}

func (s *Section) AddOption(key string, value string) *Section {
	s.Options = append(s.Options, &Option{key, value})
	return s
}

func (s *Subsection) AddOption(key string, value string) *Subsection {
	s.Options = append(s.Options, &Option{key, value})
	return s
}

func (s *Section) SetOption(key string, value string) *Section {
	for i := len(s.Options) - 1; i >= 0; i-- {
		o := s.Options[i]
		if o.IsKey(key) {
			o.Value = value
			return s
		}
	}

	return s.AddOption(key, value)
}

func (s *Subsection) SetOption(key string, value string) *Subsection {
	for i := len(s.Options) - 1; i >= 0; i-- {
		o := s.Options[i]
		if o.IsKey(key) {
			o.Value = value
			return s
		}
	}

	return s.AddOption(key, value)
}
