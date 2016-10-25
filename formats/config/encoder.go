package config

import (
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
func (e *Encoder) Encode(idx *Config) (int, error) {
	return 0, nil
}
