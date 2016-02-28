package idxfile

import (
	"crypto/sha1"
	"encoding/binary"
	"hash"
	"io"
)

type Writer struct {
	w io.Writer
	h hash.Hash
}

func NewWriter(w io.Writer) *Writer {
	h := sha1.New()

	return &Writer{
		w: io.MultiWriter(w, h),
		h: h,
	}
}

func (w *Writer) Write(idx *Idx) (int, error) {
	flow := []func(*Idx) (int, error){
		w.writeHeader,
		w.writeFanout,
		w.writeObjectsNames,
		w.writeCRC32,
		w.writeOffsets,
		w.writeChecksums,
	}

	size := 0
	for _, f := range flow {
		i, err := f(idx)
		size += i

		if err != nil {
			return size, err
		}
	}

	return size, nil
}

func (w *Writer) writeHeader(idx *Idx) (int, error) {
	count, err := w.w.Write(IdxHeader)
	if err != nil {
		return count, err
	}

	return count + 4, w.writeInt32(idx.Version)
}

func (w *Writer) writeFanout(idx *Idx) (int, error) {
	fanout := calculateFanout(idx)
	for _, c := range fanout {
		if err := w.writeInt32(c); err != nil {
			return 0, err
		}
	}

	if err := w.writeInt32(uint32(len(idx.Objects))); err != nil {
		return 0, err
	}

	return 1024, nil
}

func (w *Writer) writeObjectsNames(idx *Idx) (int, error) {
	size := 0
	for _, e := range idx.Objects {
		i, err := w.w.Write(e.Hash[:])
		size += i

		if err != nil {
			return size, err
		}
	}

	return size, nil
}

func (w *Writer) writeCRC32(idx *Idx) (int, error) {
	size := 0
	for _, e := range idx.Objects {
		i, err := w.w.Write(e.CRC32[:])
		size += i

		if err != nil {
			return size, err
		}
	}

	return size, nil
}

func (w *Writer) writeOffsets(idx *Idx) (int, error) {
	size := 0
	for _, e := range idx.Objects {
		if err := w.writeInt32(uint32(e.Offset)); err != nil {
			return size, err
		}

		size += 4

	}

	return size, nil
}

func (w *Writer) writeChecksums(idx *Idx) (int, error) {
	if _, err := w.w.Write(idx.PackfileChecksum[:]); err != nil {
		return 0, err
	}

	copy(idx.IdxChecksum[:], w.h.Sum(nil)[:20])
	if _, err := w.w.Write(idx.IdxChecksum[:]); err != nil {
		return 0, err
	}

	return 40, nil

}

func (w *Writer) writeInt32(value uint32) error {
	return binary.Write(w.w, binary.BigEndian, value)
}
