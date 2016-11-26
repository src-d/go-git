package sideband

// Type sideband type "side-band" or "side-band-64k"
type Type int8

const (
	// Sideband legacy sideband type up to 1000-byte messages
	Sideband Type = iota
	// Sideband64k sideband type up to 65519-byte messages
	Sideband64k Type = iota

	// MaxPackedSize for Sideband type
	MaxPackedSize = 1000
	// MaxPackedSize64k for Sideband64k type
	MaxPackedSize64k = 65520
)

// Channel sideband channel
type Channel byte

// Bytes returns the channel as an slice of bytes
func (ch Channel) Bytes() []byte {
	return []byte{byte(ch)}
}

const (
	// PackData packfile content
	PackData Channel = 1
	// ProgressMessage progress messages
	ProgressMessage Channel = 2
	// ErrorMessage fatal error message just before stream aborts
	ErrorMessage Channel = 3
)
