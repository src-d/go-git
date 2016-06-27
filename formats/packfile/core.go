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
	ReadVersion = ReadInt32
	// ReadCount reads and returns the count of objects field of a packfile.
	ReadCount = ReadInt32
)

// ReadInt32 reads an int32 from the packfile as Big Endian.
func ReadInt32(r io.Reader) (uint32, error) {
	var v uint32
	if err := binary.Read(r, binary.BigEndian, &v); err != nil {
		return 0, err
	}

	return v, nil
}

// IsSupportedVersion returns whether version v is supported by the parser.
// The current supported version is VersionSupported, defined above.
func IsSupportedVersion(v uint32) bool {
	return v == VersionSupported
}

// ReadSignature reads an return the signature in the packfile.
func ReadSignature(r io.Reader) ([]byte, error) {
	var sig = make([]byte, 4)
	if _, err := io.ReadFull(r, sig); err != nil {
		return []byte{}, err
	}

	return sig, nil
}

// IsValidSignature returns if sig is a valid packfile signature.
func IsValidSignature(sig []byte) bool {
	return bytes.Equal(sig, []byte{'P', 'A', 'C', 'K'})
}

// ReadHeader reads the packfile header (signature, version and object count)
// and returns the object count.
func ReadHeader(r io.Reader) (uint32, error) {
	sig, err := ReadSignature(r)
	if err != nil {
		if err == io.EOF {
			return 0, ErrEmptyPackfile
		}
		return 0, err
	}

	if !IsValidSignature(sig) {
		return 0, ErrBadSignature
	}

	ver, err := ReadVersion(r)
	if err != nil {
		return 0, err
	}

	if !IsSupportedVersion(ver) {
		return 0, ErrUnsupportedVersion.AddDetails("%d", ver)
	}

	count, err := ReadCount(r)
	if err != nil {
		return 0, err
	}

	return count, nil
}
