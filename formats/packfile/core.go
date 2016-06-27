package packfile

import (
	"bytes"
	"encoding/binary"
	"io"
)

var (
	// ErrEmptyPackfile is returned when no data is found in the packfile
	ErrEmptyPackfile = newError("empty packfile")
	// ErrBadSignature is returned when the signature in the packfile is incorrect.
	ErrBadSignature = newError("malformed pack file signature")
	// ErrUnsupportedVersion is returned by Decode when packfile version is
	// different than VersionSupported.
	ErrUnsupportedVersion = newError("unsupported packfile version")
)

const (
	// VersionSupported is the packfile version supported by this decoder.
	VersionSupported = 2
)

var (
	// ReadVersion reads and returns the version field of a packfile.
	ReadVersion = readInt32
	// ReadCount reads and returns the count of objects field of a packfile.
	ReadCount = readInt32
)

func readInt32(r io.Reader) (uint32, error) {
	var v uint32
	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return 0, err
	}

	return v, nil
}

func IsSupportedVersion(v uint32) bool {
	return v == VersionSupported
}

func ReadSignature(r io.Reader) ([]byte, error) {
	var sig = make([]byte, 4)
	if _, err := io.ReadFull(r, sig); err != nil {
		return []byte{}, err
	}

	return sig, nil
}

func IsValidSignature(sig []byte) bool {
	return bytes.Equal(sig, []byte{'P', 'A', 'C', 'K'})
}
