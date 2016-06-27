package index

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/packfile"
	"gopkg.in/src-d/go-git.v3/storage/memory"
)

var (
	// ErrMaxObjectsLimitReached is returned by Decode when the number of objects in the packfile is higher than Decoder.MaxObjectsLimit.
	ErrMaxObjectsLimitReached = newDecoderError("max. objects limit reached")
	// ErrInvalidObject is returned by Decode when an invalid object is found in the packfile.
	ErrInvalidObject = newDecoderError("invalid git object")
	// ErrPackEntryNotFound is returned by Decode when a reference in the packfile references and unknown object.
	ErrPackEntryNotFound = newDecoderError("can't find a pack entry")
	// ErrZLib is returned by Decode when there was an error unzipping the packfile contents.
	ErrZLib = newDecoderError("zlib reading error")
)

const (
	// DefaultMaxObjectsLimit is the maximum amount of objects the decoder will
	// decode before returning ErrMaxObjectsLimitReached.
	DefaultMaxObjectsLimit = 1 << 30
)

// NewFrompackfile returns a new index from a packfile reader.
func NewFromPackfile(r packfile.ByteReadReadSeeker) (Index, error) {
	count, err := packfile.ReadHeader(r)
	if err != nil {
		return nil, err
	}

	if !isValidCount(count) {
		return nil, ErrMaxObjectsLimitReached.addDetails("%d", count)
	}

	result := make(map[core.Hash]int64)

	for i := 0; i < int(count); i++ {
		offset, err := currentOffset(r)
		if err != nil {
			return nil, err
		}

		obj, err := readObject(r, result)
		if err != nil {
			return nil, err
		}

		result[obj.Hash()] = offset
	}

	return result, nil
}

func isValidCount(c uint32) bool {
	return c <= DefaultMaxObjectsLimit
}

func readObject(r packfile.ByteReadReadSeeker,
	memo map[core.Hash]int64) (core.Object, error) {

	start, err := currentOffset(r)
	if err != nil {
		return nil, err
	}
	_ = start

	var typ core.ObjectType
	var sz int64
	typ, sz, err = packfile.ReadObjectTypeAndLength(r)
	if err != nil {
		return nil, err
	}

	var cont []byte
	switch typ {
	case core.REFDeltaObject:
		cont, typ, err = readContentREFDelta(r, memo)
		sz = int64(len(cont))
	case core.OFSDeltaObject:
		cont, typ, err = readContentOFSDelta(r, start, memo)
		sz = int64(len(cont))
	case core.CommitObject, core.TreeObject, core.BlobObject, core.TagObject:
		cont, err = readContent(r)
	default:
		err = ErrInvalidObject.addDetails("tag %q", typ)
	}
	if err != nil {
		return nil, err
	}

	return memory.NewObject(typ, sz, cont), nil
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

func readContentOFSDelta(r packfile.ByteReadReadSeeker,
	objectStart int64, memo map[core.Hash]int64) (
	content []byte, typ core.ObjectType, err error) {

	_, err = currentOffset(r)
	if err != nil {
		return nil, core.ObjectType(0), err
	}

	offset, err := packfile.ReadNegativeOffset(r)
	if err != nil {
		return nil, core.ObjectType(0), err
	}

	referencedObjectOffset := objectStart + offset

	delta, err := currentOffset(r)
	if err != nil {
		return nil, core.ObjectType(0), err
	}

	r.Seek(referencedObjectOffset, os.SEEK_SET)

	refObj, err := readObject(r, memo)
	if err != nil {
		return nil, core.ObjectType(0), err
	}

	r.Seek(delta, os.SEEK_SET)

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

func readContentREFDelta(r packfile.ByteReadReadSeeker, memo map[core.Hash]int64) (
	content []byte, typ core.ObjectType, err error) {

	var ref core.Hash
	if _, err = io.ReadFull(r, ref[:]); err != nil {
		return nil, core.ObjectType(0), err
	}

	delta, err := currentOffset(r)
	if err != nil {
		return nil, core.ObjectType(0), err
	}

	refOffset, ok := memo[ref]
	if !ok {
		return nil, core.ObjectType(0), fmt.Errorf("ref %q unkown")
	}

	r.Seek(refOffset, os.SEEK_SET)

	refObj, err := readObject(r, memo)
	if err != nil {
		return nil, core.ObjectType(0), err
	}

	r.Seek(delta, os.SEEK_SET)

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
