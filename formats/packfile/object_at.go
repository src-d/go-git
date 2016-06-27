package packfile

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/storage/memory"
)

// AlreadySeener remembers already seen objects by hash or offset
// and can be asked to retrieve them. It is used to resolve
// REF-delta and OFS-delta references in the packfile.
type AlreadySeener interface {
	ByHash(hash core.Hash) (core.Object, error)
	ByOffset(offset int64) (core.Object, error)
}

type ByteReadReader interface {
	io.ByteReader
	io.Reader
}
type ByteReadReadSeeker interface {
	ByteReadReader
	io.Seeker
}

// ObjectAt returns the object at the given offset in a packfile.
func ObjectAt(packfile ByteReadReadSeeker,
	offset int64, remember AlreadySeener) (core.Object, error) {

	_, err := packfile.Seek(offset, os.SEEK_SET)
	if err != nil {
		return nil, err
	}

	typ, length, err := ReadObjectTypeAndLength(packfile)
	if err != nil {
		return nil, err
	}

	var cont []byte
	switch typ {
	case core.CommitObject, core.TreeObject, core.BlobObject, core.TagObject:
		cont, err = readContent(packfile)
	case core.REFDeltaObject:
		cont, typ, err = readContentREFDelta(packfile, remember)
		length = int64(len(cont))
	case core.OFSDeltaObject:
		cont, typ, err = readContentOFSDelta(packfile, offset, remember)
		length = int64(len(cont))
	default:
		err = fmt.Errorf("invalid object type: tag %q", typ)
	}
	if err != nil {
		return nil, err
	}

	return memory.NewObject(typ, length, cont), err
}

// Git VLQ is quite special:
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
// (0x808000) has a value of 16512 instead of zero, which means that the
// maximum 3-octet VLQ (0xffff7f) is 2113663 instead of just 2097151.
// And so forth.
//
// This is how the offset is saved in C:
//
//     dheader[pos] = ofs & 127;
//     while (ofs >>= 7)
//         dheader[--pos] = 128 | (--ofs & 127);
//
func readNegativeOffset(packfile io.Reader) (int64, error) {
	var b byte
	var err error

	if b, err = readByte(packfile); err != nil {
		return 0, err
	}

	var offset = int64(b & maskLength)
	for moreBytesInLength(b) {
		offset++
		if b, err = readByte(packfile); err != nil {
			return 0, err
		}
		offset = (offset << lengthBits) + int64(b&maskLength)
	}

	return -offset, nil
}

func readByte(r io.Reader) (byte, error) {
	buf := [1]byte{}
	if _, err := r.Read(buf[:]); err != nil {
		return 0, err
	}

	return buf[0], nil
}
