package idxfile

import (
	"io"

	"gopkg.in/src-d/go-git.v3/core"
)

type Idx struct {
	Version          uint32
	Fanout           [255]uint32
	ObjectCount      uint32
	Entries          []IdxEntry
	PackfileChecksum [20]byte
	IdxChecksum      [20]byte
}

type IdxEntry struct {
	Hash   core.Hash
	CRC32  [4]byte
	Offset uint64
}

func New(r io.Reader) (*Idx, error) {
	idx := &Idx{}

	idxReader := NewReader(r)
	_, err := idxReader.Read(idx)
	if err != nil {
		return nil, err
	}

	return idx, nil
}
