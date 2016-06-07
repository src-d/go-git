package idxfile

import (
	"crypto/sha1"
	"encoding/binary"
	"hash"
	"io"
)

// An Encoder writes idx files to an output stream.
type Encoder struct {
	io.Writer
	hash hash.Hash
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	hash := sha1.New()
	writer := io.MultiWriter(w, hash)
	return &Encoder{writer, hash}
}

// Encode writes the idx in an idx file format to the stream of the encoder.
func (e *Encoder) Encode(idx *Idxfile) (int, error) {
	flow := []func(*Idxfile) (int, error){
		e.encodeHeader,
		e.encodeFanout,
		e.encodeHashes,
		e.encodeCRC32,
		e.encodeOffsets,
		e.encodeChecksums,
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

func (e *Encoder) encodeHeader(idx *Idxfile) (int, error) {
	count, err := e.Write(idxHeader)
	if err != nil {
		return count, err
	}

	return count + 4, e.writeInt32(idx.Version)
}

func (e *Encoder) encodeFanout(idx *Idxfile) (int, error) {
	fanout := idx.calculateFanout()
	for _, c := range fanout {
		if err := e.writeInt32(c); err != nil {
			return 0, err
		}
	}

	return 1024, nil
}

func (e *Encoder) encodeHashes(idx *Idxfile) (int, error) {
	return e.encodeEntryField(idx, true)
}

func (e *Encoder) encodeCRC32(idx *Idxfile) (int, error) {
	return e.encodeEntryField(idx, false)
}

func (e *Encoder) encodeEntryField(idx *Idxfile, isHash bool) (int, error) {
	size := 0
	for _, entry := range idx.Entries {
		var data []byte
		if isHash {
			data = entry.Hash[:]
		} else {
			data = entry.CRC32[:]
		}
		i, err := e.Write(data)
		size += i

		if err != nil {
			return size, err
		}
	}

	return size, nil
}

func (e *Encoder) encodeOffsets(idx *Idxfile) (int, error) {
	size := 0
	for _, entry := range idx.Entries {
		if err := e.writeInt32(uint32(entry.Offset)); err != nil {
			return size, err
		}

		size += 4

	}

	return size, nil
}

func (e *Encoder) encodeChecksums(idx *Idxfile) (int, error) {
	if _, err := e.Write(idx.PackfileChecksum[:]); err != nil {
		return 0, err
	}

	copy(idx.IdxChecksum[:], e.hash.Sum(nil)[:20])
	if _, err := e.Write(idx.IdxChecksum[:]); err != nil {
		return 0, err
	}

	return 40, nil

}

func (e *Encoder) writeInt32(value uint32) error {
	return binary.Write(e, binary.BigEndian, value)
}
