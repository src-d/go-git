package config

import "io"

// Since git v1.8.1-rc1 config, if there are multiple
// definitions of a key, the last one wins.
// http://article.gmane.org/gmane.linux.kernel/1407184

// A Decoder reads and decodes config files from an input stream.
type Decoder struct {
	io.Reader
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r}
}

// Decode reads the whole config from its input and stores it in the
// value pointed to by config.
func (d *Decoder) Decode(config *Config) error {
	cfg := struct {
		Section struct {
				Name string
			}
	}{}
}
