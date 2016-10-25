// Package config implements decoding and encoding of git config files.
// Reference: https://git-scm.com/docs/git-config
package config

func New() *Config {
	return &Config{}
}

type Config struct {
	comment  *Comment
	sections []*Section
	includes []*Include
}

type Section struct {
	name        string
	options     []*Option
	subsections []*Subsection
}

type Subsection struct {
	name    string
	options []*Option
}

type Option struct {
	key   string
	value string
}

type Include struct {
	path   string
	config *Config
}

type Comment string

func (c *Config) Section(name string) *Section {
	for i := len(c.sections) - 1; i >= 0; i-- {
		s := c.sections[i]
		if s.name == name {
			return s
		}
	}
	s := &Section{name: name}
	c.sections = append(c.sections, s)
	return s
}

func (s *Section) Subsection(name string) *Subsection {
	for i := len(s.subsections) - 1; i >= 0; i-- {
		s := s.subsections[i]
		if s.name == name {
			return s
		}
	}
	ss := &Subsection{name: name}
	s.subsections = append(s.subsections, ss)
	return ss
}

func (s *Section) Option(key string) string {
	for i := len(s.options) - 1; i >= 0; i-- {
		o := s.options[i]
		if o.key == key {
			return o.value
		}
	}
	return ""
}

func (s *Subsection) Option(key string) string {
	for i := len(s.options) - 1; i >= 0; i-- {
		o := s.options[i]
		if o.key == key {
			return o.value
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
	s.options = append(s.options, &Option{key, value})
	return s
}

func (s *Subsection) AddOption(key string, value string) *Subsection {
	s.options = append(s.options, &Option{key, value})
	return s
}

func (s *Section) SetOption(key string, value string) *Section {
	for i := len(s.options) - 1; i >= 0; i-- {
		o := s.options[i]
		if o.key == key {
			o.value = value
			return s
		}
	}

	return s.AddOption(key, value)
}

func (s *Subsection) SetOption(key string, value string) *Subsection {
	for i := len(s.options) - 1; i >= 0; i-- {
		o := s.options[i]
		if o.key == key {
			o.value = value
			return s
		}
	}

	return s.AddOption(key, value)
}
