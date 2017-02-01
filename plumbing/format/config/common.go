package config

// New creates a new config instance.
func New() *Config {
	return &Config{}
}

// Config contains all the sections, comments and includes from a config file
type Config struct {
	Comment  *Comment
	Sections Sections
	Includes Includes
}

// Includes is a list of Includes in a config file
type Includes []*Include

// Include is reference to an included config file
type Include struct {
	Path   string
	Config *Config
}

// Comment string with out the prefix '#' or ';'
type Comment string

const (
	// NoSubsection token used to avoid provide a section to the Config.Section
	// and Config.SetSection methods
	NoSubsection = ""
)

// Section returns a existing section with the given name or creates a new one
func (c *Config) Section(name string) *Section {
	for i := len(c.Sections) - 1; i >= 0; i-- {
		s := c.Sections[i]
		if s.IsName(name) {
			return s
		}
	}

	s := &Section{Name: name}
	c.Sections = append(c.Sections, s)
	return s
}

// AddOption add an option to a given section and subsection. Use the
// NoSubsection constant for the subsection argument if no subsection is wanted.
func (c *Config) AddOption(section string, subsection string, key string, value string) *Config {
	if subsection == "" {
		c.Section(section).AddOption(key, value)
	} else {
		c.Section(section).Subsection(subsection).AddOption(key, value)
	}

	return c
}

// SetOption set an option to a given section and subsection. Use the
// NoSubsection constant for the subsection argument if no subsection is wanted.
func (c *Config) SetOption(section string, subsection string, key string, value string) *Config {
	if subsection == "" {
		c.Section(section).SetOption(key, value)
	} else {
		c.Section(section).Subsection(subsection).SetOption(key, value)
	}

	return c
}

// RemoveSection remove a section from a config file
func (c *Config) RemoveSection(name string) *Config {
	result := Sections{}
	for _, s := range c.Sections {
		if !s.IsName(name) {
			result = append(result, s)
		}
	}

	c.Sections = result
	return c
}

// RemoveSubsection remove a subsection from a config file
func (c *Config) RemoveSubsection(section string, subsection string) *Config {
	for _, s := range c.Sections {
		if s.IsName(section) {
			result := Subsections{}
			for _, ss := range s.Subsections {
				if !ss.IsName(subsection) {
					result = append(result, ss)
				}
			}
			s.Subsections = result
		}
	}

	return c
}
