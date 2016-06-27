package index

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/packfile"
)

var (
	// ErrInvalidObject is returned by Decode when an invalid object is
	// found in the packfile.
	ErrInvalidObject = packfile.NewError("invalid git object")

	// ErrPackEntryNotFound is returned by Decode when a reference in
	// the packfile references and unknown object.
	ErrPackEntryNotFound = packfile.NewError("can't find a pack entry")

	// ErrZLib is returned by Decode when there was an error unzipping
	// the packfile contents.
	ErrZLib = packfile.NewError("zlib reading error")
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
		return nil, packfile.ErrMaxObjectsLimitReached.AddDetails("%d", count)
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
