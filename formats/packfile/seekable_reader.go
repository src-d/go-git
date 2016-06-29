package packfile

import (
	"io"
	"os"

	"gopkg.in/src-d/go-git.v3/core"
)

// SeekableReader implements Reader from a packfile in a io.ReadSeeker.
type SeekableReader struct {
	io.ReadSeeker
	OffsetsByHash map[core.Hash]int64
}

func NewSeekableReader(r io.ReadSeeker) *SeekableReader {
	return &SeekableReader{
		r,
		make(map[core.Hash]int64),
	}
}

func (r *SeekableReader) Read(p []byte) (int, error) {
	return r.ReadSeeker.Read(p)
}

func (r *SeekableReader) ReadByte() (byte, error) {
	var p [1]byte
	_, err := r.ReadSeeker.Read(p[:])
	if err != nil {
		return 0, err
	}

	return p[0], nil
}

func (r *SeekableReader) Offset() (int64, error) {
	return r.Seek(0, os.SEEK_CUR)
}

func (r *SeekableReader) Remember(o int64, obj core.Object) error {
	h := obj.Hash()
	if _, ok := r.OffsetsByHash[h]; ok {
		return ErrDuplicatedObj.AddDetails("with hash %s", h)
	}

	r.OffsetsByHash[h] = o

	return nil
}

func (r *SeekableReader) RecallByHash(h core.Hash) (core.Object, error) {
	o, ok := r.OffsetsByHash[h]
	if !ok {
		return nil, ErrCannotRecall.AddDetails("hash not found: %s", h)
	}

	return r.RecallByOffset(o)
}

func (r *SeekableReader) RecallByOffset(o int64) (obj core.Object, err error) {
	beforeJump, err := r.Offset()
	if err != nil {
		return nil, err
	}

	defer func() {
		_, seekErr := r.Seek(beforeJump, os.SEEK_SET)
		if err == nil {
			err = seekErr
		}
	}()

	// jump to offset o
	_, err = r.Seek(o, os.SEEK_SET)
	if err != nil {
		return nil, err
	}

	return NewParser(r).ReadObject()
}
