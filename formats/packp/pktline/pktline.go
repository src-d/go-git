// Package pktline implements reading and creating pkt-lines as per
// https://github.com/git/git/blob/master/Documentation/technical/protocol-common.txt.
package pktline

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// MaxPayloadSize is the maximum payload size of a pkt-line in bytes.
	MaxPayloadSize = 65516
)

var (
	flush = []byte{'0', '0', '0', '0'}
)

// PktLine values represent a succession of pkt-lines.
// Values from this type are not zero-value safe, see the functions New
// and NewFromString below.
type PktLine struct {
	io.Reader
}

// ErrPayloadTooLong is returned by New and NewFromString when any of
// the provided payloads is bigger than MaxPayloadSize.
var ErrPayloadTooLong = errors.New("payload is too long")

// New returns the concatenation of several pkt-lines, each of them with
// the payload specified by the contents of each input byte slice.  An
// empty payload byte slice will produce a flush-pkt.
func New(payloads ...[]byte) (PktLine, error) {
	ret := []io.Reader{}
	for _, p := range payloads {
		if err := add(&ret, p); err != nil {
			return PktLine{}, err
		}
	}

	return PktLine{io.MultiReader(ret...)}, nil
}

func add(dst *[]io.Reader, e []byte) error {
	if len(e) > MaxPayloadSize {
		return ErrPayloadTooLong
	}

	if len(e) == 0 {
		*dst = append(*dst, bytes.NewReader(flush))
		return nil
	}

	n := len(e) + 4
	*dst = append(*dst, strings.NewReader(fmt.Sprintf("%04x", n)))
	*dst = append(*dst, bytes.NewReader(e))

	return nil
}

// NewFromStrings returns the concatenation of several pkt-lines, each
// of them with the payload specified by the contents of each input
// string.  An empty payload string will produce a flush-pkt.
func NewFromStrings(payloads ...string) (PktLine, error) {
	ret := []io.Reader{}
	for _, p := range payloads {
		if err := addString(&ret, p); err != nil {
			return PktLine{}, err
		}
	}

	return PktLine{io.MultiReader(ret...)}, nil
}

func addString(dst *[]io.Reader, s string) error {
	if len(s) > MaxPayloadSize {
		return ErrPayloadTooLong
	}

	if len(s) == 0 {
		*dst = append(*dst, bytes.NewReader(flush))
		return nil
	}

	n := len(s) + 4
	*dst = append(*dst, strings.NewReader(fmt.Sprintf("%04x", n)))
	*dst = append(*dst, strings.NewReader(s))

	return nil
}
