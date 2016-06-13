package packfile

import (
	"bytes"
	"compress/zlib"
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

// ObjectAt returns the object at the given offset in a packfile.
func ObjectAt(packfile io.ReadSeeker,
	offset int64, remember AlreadySeener) (core.Object, error) {

	_, err := packfile.Seek(offset, os.SEEK_SET)
	if err != nil {
		return nil, err
	}

	typ, length, err := readTypeAndLength(packfile)
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

func readTypeAndLength(packfile io.Reader) (core.ObjectType, int64, error) {
	var buf [1]byte
	if _, err := packfile.Read(buf[:]); err != nil {
		return core.ObjectType(0), 0, err
	}

	typ := parseType(buf[0])

	length, err := readLength(buf[0], packfile)

	return typ, length, err
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
func readLength(first byte, packfile io.Reader) (int64, error) {
	length := int64(first & maskFirstLength)

	buf := [1]byte{first}
	shift := firstLengthBits
	for moreBytesInLength(buf[0]) {
		if _, err := packfile.Read(buf[:]); err != nil {
			return 0, err
		}

		length += int64(buf[0]&maskLength) << shift
		shift += lengthBits
	}

	return length, nil
}

func moreBytesInLength(b byte) bool {
	return b&maskContinue > 0
}

func readContent(packfile io.Reader) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := inflate(packfile, buf)

	return buf.Bytes(), err
}

func readContentREFDelta(packfile io.Reader, remember AlreadySeener) (content []byte,
	typ core.ObjectType, err error) {

	var ref core.Hash
	if _, err = io.ReadFull(packfile, ref[:]); err != nil {
		return nil, core.ObjectType(0), err
	}

	diff := bytes.NewBuffer(nil)
	if err = inflate(packfile, diff); err != nil {
		return nil, core.ObjectType(0), err
	}

	referenced, err := remember.ByHash(ref)
	if err != nil {
		return nil, core.ObjectType(0), fmt.Errorf("reference not found: %s", ref)
	}

	content = PatchDelta(referenced.Content(), diff.Bytes())
	if content == nil {
		return nil, core.ObjectType(0), fmt.Errorf("patching error: %q", ref)
	}

	return content, referenced.Type(), nil
}

func inflate(r io.Reader, w io.Writer) (err error) {
	zr, err := zlib.NewReader(r)
	if err != nil {
		if err != zlib.ErrHeader {
			return fmt.Errorf("zlib reading error: %s", err)
		}
	}

	defer func() {
		closeErr := zr.Close()
		if err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(w, zr)
	return err
}

func readContentOFSDelta(packfile io.Reader,
	objectStart int64, remember AlreadySeener) (content []byte,
	typ core.ObjectType, err error) {

	offset, err := readNegativeOffset(packfile)
	if err != nil {
		return nil, core.ObjectType(0), err
	}

	diff := bytes.NewBuffer(nil)
	if err = inflate(packfile, diff); err != nil {
		return nil, core.ObjectType(0), err
	}

	referenced, err := remember.ByOffset(objectStart + offset)
	if err != nil {
		return nil, core.ObjectType(0), err
	}

	patched := PatchDelta(referenced.Content(), diff.Bytes())
	if patched == nil {
		return nil, core.ObjectType(0), fmt.Errorf("paching error")
	}

	return patched, referenced.Type(), nil
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
