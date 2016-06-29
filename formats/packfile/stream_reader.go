package packfile

import (
	"io"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/packfile/internal/readcounter"
)

// StreamReader implements Reader from a packfile in a io.Reader.
type StreamReader struct {
	byOffset    map[int64]core.Object
	byHash      map[core.Hash]core.Object
	readCounter *readcounter.ReadCounter
}

func NewStreamReader(r io.Reader) *StreamReader {
	return &StreamReader{
		byHash:      make(map[core.Hash]core.Object, 0),
		byOffset:    make(map[int64]core.Object, 0),
		readCounter: readcounter.New(r),
	}
}

func (r *StreamReader) Read(p []byte) (int, error) {
	return r.readCounter.Read(p)
}

func (r *StreamReader) ReadByte() (byte, error) {
	return r.readCounter.ReadByte()
}

func (r *StreamReader) Offset() (int64, error) {
	return r.readCounter.Count(), nil
}

func (r *StreamReader) Remember(o int64, obj core.Object) error {
	h := obj.Hash()
	if _, ok := r.byHash[h]; ok {
		return ErrDuplicatedObj.AddDetails("with hash: %s", h)
	}
	if _, ok := r.byOffset[o]; ok {
		return ErrDuplicatedObj.AddDetails("at offset: %d", o)
	}
	r.byHash[h] = obj
	r.byOffset[o] = obj

	return nil
}

func (r *StreamReader) RecallByHash(h core.Hash) (core.Object, error) {
	obj, ok := r.byHash[h]
	if !ok {
		return nil, ErrCannotRecall.AddDetails("hash not found: %s", h)
	}

	return obj, nil
}

func (r *StreamReader) RecallByOffset(o int64) (core.Object, error) {
	obj, ok := r.byOffset[o]
	if !ok {
		return nil, ErrCannotRecall.AddDetails("no object found at offset %d", o)
	}

	return obj, nil
}
