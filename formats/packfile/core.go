package packfile

import (
	"bytes"
	"encoding/binary"
	"io"

	"gopkg.in/src-d/go-git.v3/core"
)

var (
	// ErrEmptyPackfile is returned when no data is found in the packfile
	ErrEmptyPackfile = NewError("empty packfile")
	// ErrBadSignature is returned when the signature in the packfile is incorrect.
	ErrBadSignature = NewError("malformed pack file signature")
	// ErrUnsupportedVersion is returned by Decode when packfile version is
	// different than VersionSupported.
	ErrUnsupportedVersion = NewError("unsupported packfile version")
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

// ReadObjectTypeAndLength reads and return an the object type and the length field
// from an object entry in a packfile.
func ReadObjectTypeAndLength(r io.ByteReader) (core.ObjectType, int64, error) {
	t, c, err := readType(r)
	if err != nil {
		return t, 0, err
	}

	l, err := readLength(c, r)

	return t, l, err
}

func readType(r io.ByteReader) (core.ObjectType, byte, error) {
	var c byte
	var err error
	if c, err = r.ReadByte(); err != nil {
		return core.ObjectType(0), 0, err
	}
	typ := parseType(c)

	return typ, c, nil
}

var (
	maskContinue    = uint8(128) // 1000 0000
	maskType        = uint8(112) // 0111 0000
	maskFirstLength = uint8(15)  // 0000 1111
	firstLengthBits = uint8(4)   // the first byte has 4 bits to store the length
	maskLength      = uint8(127) // 0111 1111
	lengthBits      = uint8(7)   // subsequent bytes has 7 bits to store the length
)

func parseType(b byte) core.ObjectType {
	return core.ObjectType((b & maskType) >> firstLengthBits)
}

// Reads the last 4 bits from the first byte in the object.
// If more bytes are required for the length, read more bytes
// and use the first 7 bits of each one until no more bytes
// are required.
func readLength(first byte, r io.ByteReader) (int64, error) {
	length := int64(first & maskFirstLength)

	c := first
	shift := firstLengthBits
	var err error
	for moreBytesInLength(c) {
		if c, err = r.ReadByte(); err != nil {
			return 0, err
		}

		length += int64(c&maskLength) << shift
		shift += lengthBits
	}

	return length, nil
}

func moreBytesInLength(c byte) bool {
	return c&maskContinue > 0
}

// ReadNegativeOffset reads and returns an offset from a OFS DELTA
// object entry in a packfile. OFS DELTA offsets are specified in Git
// VLQ special format:
//
// Ordinary VLQ has some redundancies, example:  the number 358 can be
// encoded as the 2-octet VLQ 0x8166 or the 3-octet VLQ 0x808166 or the
// 4-octet VLQ 0x80808166 and so forth.
//
// To avoid these redundancies, the VLQ format used in Git removes this
// prepending redundancy and extends the representable range of shorter
// VLQs by adding an offset to VLQs of 2 or more octets in such a way
// that the lowest possible value for such an (N+1)-octet VLQ becomes
// exactly one more than the maximum possible value for an N-octet VLQ.
// In particular, since a 1-octet VLQ can store a maximum value of 127,
// the minimum 2-octet VLQ (0x8000) is assigned the value 128 instead of
// 0. Conversely, the maximum value of such a 2-octet VLQ (0xff7f) is
// 16511 instead of just 16383. Similarly, the minimum 3-octet VLQ
//       (0x808000) has a value of 16512 instead of zero, which means
//       that the maximum 3-octet VLQ (0xffff7f) is 2113663 instead of
//       just 2097151.  And so forth.
//
// This is how the offset is saved in C:
//
//     dheader[pos] = ofs & 127; while (ofs >>= 7) dheader[--pos] = 128
//     | (--ofs & 127);
//
func ReadNegativeOffset(r io.ByteReader) (int64, error) {
	var c byte
	var err error

	if c, err = r.ReadByte(); err != nil {
		return 0, err
	}

	var offset = int64(c & maskLength)
	for moreBytesInLength(c) {
		offset++
		if c, err = r.ReadByte(); err != nil {
			return 0, err
		}
		offset = (offset << lengthBits) + int64(c&maskLength)
	}

	return -offset, nil
}
