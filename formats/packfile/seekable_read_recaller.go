package packfile

import (
	"io"
	"os"

	"gopkg.in/src-d/go-git.v3/core"
)

// SeekableReadRecaller implements ReadRecaller from a packfile in a
// io.ReadSeeker.  Remembering does not actually stores a reference to
// the objects; the object offset is remebered instead and the packfile
// is read again everytime a recall operation is requested. This saves
// memory buy can be very slow if the associated io.ReadSeeker is slow
// (like a hard disk).
type SeekableReadRecaller struct {
	io.ReadSeeker
	OffsetsByHash map[core.Hash]int64
}

// NewSeekableReadRecaller returns a new SeekableReadRecaller that reads
// form r.
func NewSeekableReadRecaller(r io.ReadSeeker) *SeekableReadRecaller {
	return &SeekableReadRecaller{
		r,
		make(map[core.Hash]int64),
	}
}

// Read reads up to len(p) bytes into p.
func (r *SeekableReadRecaller) Read(p []byte) (int, error) {
	return r.ReadSeeker.Read(p)
}

// ReadByte reads a byte.
func (r *SeekableReadRecaller) ReadByte() (byte, error) {
	var p [1]byte
	_, err := r.ReadSeeker.Read(p[:])
	if err != nil {
		return 0, err
	}

	return p[0], nil
}

// Offset returns the offset for the next Read or ReadByte.
func (r *SeekableReadRecaller) Offset() (int64, error) {
	return r.Seek(0, os.SEEK_CUR)
}

// Remember stores the offset of the object and its hash, but not the object itself.
func (r *SeekableReadRecaller) Remember(o int64, obj core.Object) error {
	h := obj.Hash()
	if _, ok := r.OffsetsByHash[h]; ok {
		return ErrDuplicatedObj.AddDetails("with hash %s", h)
	}

	r.OffsetsByHash[h] = o

	return nil
}

// RecallByHash returns the object for a given hash by looking for it again in
// the io.ReadeSeerker.
func (r *SeekableReadRecaller) RecallByHash(h core.Hash) (core.Object, error) {
	o, ok := r.OffsetsByHash[h]
	if !ok {
		return nil, ErrCannotRecall.AddDetails("hash not found: %s", h)
	}

	return r.RecallByOffset(o)
}

// RecallByOffset returns the object for a given offset by looking for it again in
// the io.ReadeSeerker.
func (r *SeekableReadRecaller) RecallByOffset(o int64) (obj core.Object, err error) {
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
