package pktline

import (
	"errors"
	"io"
)

const (
	lenSize = 4
)

// ErrInvalidPktLen is returned by Err() when an invalid pkt-len is found.
var ErrInvalidPktLen = errors.New("invalid pkt-len found")

// Scanner provides a convenient interface for reading the payloads of a
// series of pkt-lines.  It takes an io.Reader providing the source,
// which then can be tokenized through repeated calls to the Scan
// method.
//
// After each Scan call, the Bytes method will return the payload of the
// corresponding pkt-line on a shared buffer, which will be 65516 bytes
// or smaller.  Flush pkt-lines are represented by empty byte slices.
//
// Scanning stops at EOF or the first I/O error.
type Scanner struct {
	r       io.Reader     // The reader provided by the client
	err     error         // Sticky error
	payload []byte        // Last pkt-payload
	len     [lenSize]byte // Last pkt-len
}

// NewScanner returns a new Scanner to read from r.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{
		r: r,
	}
}

// Err returns the first error encountered by the Scanner.
func (s *Scanner) Err() error {
	return s.err
}

// Scan advances the Scanner to the next pkt-line, whose payload will
// then be available through the Bytes method.  Scanning stops at EOF
// or the first I/O error.  After Scan returns false, the Err method
// will return any error that occurred during scanning, except that if
// it was io.EOF, Err will return nil.
func (s *Scanner) Scan() bool {
	var l int
	l, s.err = s.readPayloadLen()
	if s.err == io.EOF {
		s.err = nil
		return false
	}
	if s.err != nil {
		return false
	}

	if cap(s.payload) < l {
		s.payload = make([]byte, 0, l)
	}

	if _, s.err = io.ReadFull(s.r, s.payload[:l]); s.err != nil {
		return false
	}
	s.payload = s.payload[:l]

	return true
}

// Bytes returns the most recent payload generated by a call to Scan.
// The underlying array may point to data that will be overwritten by a
// subsequent call to Scan. It does no allocation.
func (s *Scanner) Bytes() []byte {
	return s.payload
}

// Method readPayloadLen returns the payload length by reading the
// pkt-len and substracting the pkt-len size.
func (s *Scanner) readPayloadLen() (int, error) {
	if _, err := io.ReadFull(s.r, s.len[:]); err != nil {
		if err == io.ErrUnexpectedEOF {
			return 0, ErrInvalidPktLen
		}

		return 0, err
	}

	n, err := hexDecode(s.len)
	if err != nil {
		return 0, err
	}

	switch {
	case n == 0:
		return 0, nil
	case n <= lenSize:
		return 0, ErrInvalidPktLen
	case n > MaxPayloadSize+lenSize:
		return 0, ErrInvalidPktLen
	default:
		return n - lenSize, nil
	}
}

// Turns the hexadecimal representation of a number in a byte slice into
// a number. This function substitute strconv.ParseUint(string(buf), 16,
// 16) and/or hex.Decode, to avoid generating new strings, thus helping the
// GC.
func hexDecode(buf [lenSize]byte) (int, error) {
	var ret int
	for i := 0; i < lenSize; i++ {
		n, err := asciiHexToByte(buf[i])
		if err != nil {
			return 0, ErrInvalidPktLen
		}
		ret = 16*ret + int(n)
	}
	return ret, nil
}

// turns the hexadecimal ascii representation of a byte into its
// numerical value.  Example: from 'b' to 11 (0xb).
func asciiHexToByte(b byte) (byte, error) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', nil
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, nil
	default:
		return 0, ErrInvalidPktLen
	}
}
