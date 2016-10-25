package config

import (
	"fmt"
	"io"
)

// An Encoder writes config files to an output stream.
type Encoder struct {
	io.Writer
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w}
}

// Encode writes the config in git config format to the stream of the encoder.
func (e *Encoder) Encode(cfg *Config) error {
	for _, s := range cfg.sections {
		if len(s.options) > 0 {
			fmt.Fprintf(e, "[%s]\n", s.name)
			for _, o := range s.options {
				fmt.Fprintf(e, "\t%s = %s\n", o.key, o.value)
			}
		}
		for _, ss := range s.subsections {
			if len(ss.options) > 0 {
				//TODO: escape
				fmt.Fprintf(e, "[%s \"%s\"]\n", s.name, ss.name)
				for _, o := range ss.options {
					fmt.Fprintf(e, "\t%s = %s\n", o.key, o.value)
				}
			}
		}
	}
	return nil
}
