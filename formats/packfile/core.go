package packfile

import (
	"encoding/binary"
	"io"
)

var (
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
	// ReadVersion reads and returns the count of objects field of a packfile.
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
