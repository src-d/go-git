package index

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/packfile"
	"gopkg.in/src-d/go-git.v3/storage/memory"
)

var (
	// ErrEmptyPackfile is returned by Decode when no data is found in the packfile
	ErrEmptyPackfile = newDecoderError("empty packfile")
	// ErrUnsupportedVersion is returned by Decode when packfile version is different than VersionSupported.
	ErrUnsupportedVersion = newDecoderError("unsupported packfile version")
	// ErrMaxObjectsLimitReached is returned by Decode when the number of objects in the packfile is higher than Decoder.MaxObjectsLimit.
	ErrMaxObjectsLimitReached = newDecoderError("max. objects limit reached")
	// ErrMalformedPackfile is returned by Decode when the packfile is corrupt.
	ErrMalformedPackfile = newDecoderError("malformed pack file, does not start with 'PACK'")
	// ErrInvalidObject is returned by Decode when an invalid object is found in the packfile.
	ErrInvalidObject = newDecoderError("invalid git object")
	// ErrPackEntryNotFound is returned by Decode when a reference in the packfile references and unknown object.
	ErrPackEntryNotFound = newDecoderError("can't find a pack entry")
	// ErrZLib is returned by Decode when there was an error unzipping the packfile contents.
	ErrZLib = newDecoderError("zlib reading error")
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
	count, err := readHeader(r)
	if err != nil {
		return nil, err
	}

	result := make(map[core.Hash]int64)

	fmt.Printf("%d objects in the packfile\n", count)

	for i := 0; i < int(count); i++ {

		fmt.Printf("trying to read object %d:\n", i)

		offset, err := currentOffset(r)
		if err != nil {
			return nil, err
		}
		fmt.Printf("\tcurrent offset = %d:\n", offset)

		obj, err := readObject(r)
		if err != nil {
			return nil, err
		}

		result[obj.Hash()] = offset
		fmt.Printf("\tobject hash : %s\n", obj.Hash())
	}

	return result, nil
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

func readObject(r io.ReadSeeker) (core.Object, error) {

	start, err := currentOffset(r)
	if err != nil {
		return nil, err
	}
	_ = start

	var typ core.ObjectType
	var sz int64
	typ, sz, err = readTypeAndLength(r)
	if err != nil {
		return nil, err
	}
	fmt.Printf("\treaded size (from object head): %d\n", sz)

	var cont []byte
	fmt.Printf("\tobject type: %s\n", typ)
	offset, _ := currentOffset(r)
	fmt.Printf("\tcurrent offset = %d:\n", offset)
	switch typ {
	case core.REFDeltaObject:
		err = fmt.Errorf("REFDeltaObject not supported yet")
		_ = cont
		/*
			cont, typ, err = readContentREFDelta(d.readCounter, d)
			sz = int64(len(cont))
		*/
	case core.OFSDeltaObject:
		err = fmt.Errorf("OFSDeltaObject not supported yet")
		cont, typ, err = readContentOFSDelta(r, start)
		sz = int64(len(cont))
	case core.CommitObject, core.TreeObject, core.BlobObject, core.TagObject:
		cont, err = readContent(r)
		fmt.Printf("\treaded size (unzipped): %d\n", len(cont))
		offset, _ := currentOffset(r)
		fmt.Printf("\tcurrent offset = %d:\n", offset)
	default:
		err = ErrInvalidObject.addDetails("tag %q", typ)
	}
	if err != nil {
		return nil, err
	}

	return memory.NewObject(typ, sz, cont), nil
}

func readTypeAndLength(r io.Reader) (core.ObjectType, int64, error) {
	var buf [1]byte
	if _, err := r.Read(buf[:]); err != nil {
		return core.ObjectType(0), 0, err
	}

	typ := parseType(buf[0])
	length, err := readLength(buf[0], r)

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

func readContent(r io.Reader) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := inflate(r, buf)

	return buf.Bytes(), err
}

// https://github.com/golang/go/commit/7ba54d45732219af86bde9a5b73c145db82b70c6
// https://groups.google.com/forum/#!topic/golang-nuts/fWTRdHpt0QI
// https://gowalker.org/compress/zlib#NewReader
type byteReader struct {
	io.Reader
}

func (b *byteReader) ReadByte() (byte, error) {
	var p [1]byte
	_, err := b.Read(p[:])
	if err != nil {
		return 0, err
	}

	return p[0], nil
}

func inflate(r io.Reader, w io.Writer) (err error) {
	byteReader := &byteReader{r} // see byteReader comments above
	zr, err := zlib.NewReader(byteReader)
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

// DecoderError specifies errors returned by Decode.
type DecoderError struct {
	reason, details string
}

func newDecoderError(reason string) *DecoderError {
	return &DecoderError{reason: reason}
}

func (e *DecoderError) Error() string {
	if e.details == "" {
		return e.reason
	}

	return fmt.Sprintf("%s: %s", e.reason, e.details)
}

func (e *DecoderError) addDetails(format string, args ...interface{}) *DecoderError {
	return &DecoderError{
		reason:  e.reason,
		details: fmt.Sprintf(format, args...),
	}
}

func readContentOFSDelta(r io.ReadSeeker, objectStart int64) (
	content []byte, typ core.ObjectType, err error) {

	_, err = currentOffset(r)
	if err != nil {
		return nil, core.ObjectType(0), err
	}

	offset, err := readNegativeOffset(r)
	if err != nil {
		return nil, core.ObjectType(0), err
	}

	referencedObjectOffset := objectStart + offset

	fmt.Printf("\tOFS DELTA offset = %d\n", offset)
	fmt.Printf("\tOFS DELTA destination of jump = %d\n", referencedObjectOffset)

	delta, err := currentOffset(r)
	if err != nil {
		return nil, core.ObjectType(0), err
	}
	fmt.Printf("\tOFS delta data at = %d\n", delta)

	r.Seek(referencedObjectOffset, os.SEEK_SET)
	current, _ := currentOffset(r)
	fmt.Printf("\tjust jumped to referenced object at %d\n", current)

	refObj, err := readObject(r)
	if err != nil {
		return nil, core.ObjectType(0), err
	}
	fmt.Printf("\treaded type of referenced object: %s\n", refObj.Type())

	r.Seek(delta, os.SEEK_SET)
	current, _ = currentOffset(r)
	fmt.Printf("\tjumped back to %d (delta data)\n", current)

	diff := bytes.NewBuffer(nil)
	if err = inflate(r, diff); err != nil {
		return nil, core.ObjectType(0), err
	}

	patched := packfile.PatchDelta(refObj.Content(), diff.Bytes())
	if patched == nil {
		return nil, core.ObjectType(0), fmt.Errorf("paching error")
	}

	return patched, refObj.Type(), nil
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
func readNegativeOffset(r io.Reader) (int64, error) {
	var b byte
	var err error

	if b, err = readByte(r); err != nil {
		return 0, err
	}

	var offset = int64(b & maskLength)
	for moreBytesInLength(b) {
		offset++
		if b, err = readByte(r); err != nil {
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

func currentOffset(r io.Seeker) (int64, error) {
	return r.Seek(0, os.SEEK_CUR)
}
