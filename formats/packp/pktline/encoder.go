package pktline

import (
	"bytes"
	"io"
)

// An Encoder writes pkt-lines to an output stream.
type Encoder struct {
	w io.Writer
}

const (
	// MaxPayloadSize is the maximum payload size of a pkt-line in bytes.
	MaxPayloadSize = 65516
)

var (
	// FlushPkt are the contents of a flush-pkt pkt-line.
	FlushPkt = []byte{'0', '0', '0', '0'}
	// Flush is the payload to use with the Encode method to encode a flush-pkt.
	Flush = []byte{}
	// FlushString is the payload to use with the EncodeString method to encode a flush-pkt.
	FlushString = ""
)

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

// Flush encodes a flush-pkt to the output stream.
func (e *Encoder) Flush() error {
	_, err := e.w.Write(FlushPkt)
	return err
}

// Encode encodes a pkt-line with the payload specified and write it to
// the output stream.  If several payloads are specified, each of them
// will get streamed in their own pkt-lines.
func (e *Encoder) Encode(payloads ...[]byte) error {
	for _, p := range payloads {
		if err := checkPayloadLength(len(p)); err != nil {
			return err
		}

		if bytes.Equal(p, Flush) {
			if err := e.Flush(); err != nil {
				return err
			}
			continue
		}

		n := len(p) + 4
		if _, err := e.w.Write(asciiHex16(n)); err != nil {
			return err
		}
		if _, err := e.w.Write(p); err != nil {
			return err
		}
	}

	return nil
}

// EncodeString works similarly as Encode but payloads are specified as strings.
func (e *Encoder) EncodeString(payloads ...string) error {
	for _, p := range payloads {
		if err := e.Encode([]byte(p)); err != nil {
			return err
		}
	}

	return nil
}

// Returns the hexadecimal ascii representation of the 16 less
// significant bits of n.  The length of the returned slice will always
// be 4.  Example: if n is 1234 (0x4d2), the return value will be
// []byte{'0', '4', 'd', '2'}.
func asciiHex16(n int) []byte {
	var ret [4]byte
	ret[0] = byteToASCIIHex(byte(n & 0xf000 >> 12))
	ret[1] = byteToASCIIHex(byte(n & 0x0f00 >> 8))
	ret[2] = byteToASCIIHex(byte(n & 0x00f0 >> 4))
	ret[3] = byteToASCIIHex(byte(n & 0x000f))

	return ret[:]
}

// turns a byte into its hexadecimal ascii representation.  Example:
// from 11 (0xb) to 'b'.
func byteToASCIIHex(n byte) byte {
	if n < 10 {
		return '0' + n
	}

	return 'a' - 10 + n
}
