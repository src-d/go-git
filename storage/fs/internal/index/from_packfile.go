package index

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var (
	// ErrEmptyPackfile is returned by Decode when no data is found in the packfile
	ErrEmptyPackfile = fmt.Errorf("empty packfile")
	// ErrUnsupportedVersion is returned by Decode when packfile version is different than VersionSupported.
	ErrUnsupportedVersion = fmt.Errorf("unsupported packfile version")
	// ErrMaxObjectsLimitReached is returned by Decode when the number of objects in the packfile is higher than Decoder.MaxObjectsLimit.
	ErrMaxObjectsLimitReached = fmt.Errorf("max. objects limit reached")
	// ErrMalformedPackfile is returned by Decode when the packfile is corrupt.
	ErrMalformedPackfile = fmt.Errorf("malformed pack file, does not start with 'PACK'")
	// ErrInvalidObject is returned by Decode when an invalid object is found in the packfile.
	ErrInvalidObject = fmt.Errorf("invalid git object")
	// ErrPackEntryNotFound is returned by Decode when a reference in the packfile references and unknown object.
	ErrPackEntryNotFound = fmt.Errorf("can't find a pack entry")
	// ErrZLib is returned by Decode when there was an error unzipping the packfile contents.
	ErrZLib = fmt.Errorf("zlib reading error")
)

const (
	// DefaultMaxObjectsLimit is the maximum amount of objects the decoder will decode before
	// returning ErrMaxObjectsLimitReached.
	DefaultMaxObjectsLimit = 1 << 20
	// VersionSupported is the packfile version supported by this decoder.
	VersionSupported = 2
)

// NewFrompackfile returns a new index from a packfile reader.
func NewFromPackfile(r io.ReadSeeker) (Index, error) {
	_, err := readHeader(r)
	if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("TODO")
}

func readHeader(r io.Reader) (uint32, error) {
	sig, err := readSignature(r)
	if err != nil {
		if err == io.EOF {
			return 0, ErrEmptyPackfile
		}
		return 0, err
	}

	if !isValidSignature(sig) {
		return 0, ErrMalformedPackfile
	}

	ver, err := readVersion(r)
	if err != nil {
		return 0, err
	}

	if !isSupportedVersion(ver) {
		return 0, ErrUnsupportedVersion
	}

	count, err := readCount(r)
	if err != nil {
		return 0, err
	}

	if !isValidCount(count) {
		return 0, ErrMaxObjectsLimitReached
	}

	return count, nil
}

func readSignature(r io.Reader) ([]byte, error) {
	var sig = make([]byte, 4)
	if _, err := io.ReadFull(r, sig); err != nil {
		return []byte{}, err
	}

	return sig, nil
}

func isValidSignature(sig []byte) bool {
	return bytes.Equal(sig, []byte{'P', 'A', 'C', 'K'})
}

func readVersion(r io.Reader) (uint32, error) {
	var v uint32
	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return 0, err
	}

	return v, nil
}

func isSupportedVersion(v uint32) bool {
	return v == VersionSupported
}

func readCount(r io.Reader) (uint32, error) {
	var c uint32
	if err := binary.Read(r, binary.BigEndian, &c); err != nil {
		return 0, err
	}

	return c, nil
}

func isValidCount(c uint32) bool {
	return c <= DefaultMaxObjectsLimit
}
