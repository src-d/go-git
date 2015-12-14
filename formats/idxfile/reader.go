package idxfile

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v2/core"
)

const (
	IdxVersionSupported = 2
)

var (
	IdxHeader             = []byte{255, 't', 'O', 'c'}
	UnsupportedVersionErr = errors.New("Unsuported version")
	MalformedIdxFileErr   = errors.New("Malformed IDX file")
)

type Idx struct {
	Version          uint32
	Fanout           [255]uint32
	ObjectCount      uint32
	Objects          []IdxEntry
	PackfileChecksum [20]byte
	IdxChecksum      [20]byte
}

type IdxEntry struct {
	Hash   core.Hash
	CRC32  [4]byte
	Offset uint64
}

type Reader struct {
	r io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

func (r *Reader) Read(idx *Idx) (int64, error) {
	if err := r.validateHeader(); err != nil {
		return -1, err
	}

	flow := []func(*Idx) error{
		r.readVersion,
		r.readFanout,
		r.readObjectNames,
		r.readCRC32,
		r.readOffsets,
		r.readChecksums,
	}

	for _, f := range flow {
		if err := f(idx); err != nil {
			return -1, err
		}
	}

	if !r.isValid(idx) {
		return -1, MalformedIdxFileErr
	}

	return 0, nil
}

func (r *Reader) validateHeader() error {
	var header = make([]byte, 4)
	if _, err := r.r.Read(header); err != nil {
		return err
	}

	if !bytes.Equal(header, IdxHeader) {
		return MalformedIdxFileErr
	}

	return nil
}

func (r *Reader) readVersion(idx *Idx) error {
	version, err := r.readInt32()
	if err != nil {
		return err
	}

	if version > IdxVersionSupported {
		return UnsupportedVersionErr
	}

	idx.Version = version

	return nil
}

func (r *Reader) readFanout(idx *Idx) error {
	for i := 0; i < 255; i++ {
		var err error
		idx.Fanout[i], err = r.readInt32()
		if err != nil {
			return err
		}
	}

	var err error
	idx.ObjectCount, err = r.readInt32()
	if err != nil {
		return err
	}

	return nil
}

func (r *Reader) readObjectNames(idx *Idx) error {
	count := int(idx.ObjectCount)
	for i := 0; i < count; i++ {
		var ref core.Hash
		if _, err := r.r.Read(ref[:]); err != nil {
			return err
		}

		idx.Objects = append(idx.Objects, IdxEntry{Hash: ref})
	}

	return nil
}

func (r *Reader) readCRC32(idx *Idx) error {
	count := int(idx.ObjectCount)
	for i := 0; i < count; i++ {
		if _, err := r.r.Read(idx.Objects[i].CRC32[:]); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reader) readOffsets(idx *Idx) error {
	count := int(idx.ObjectCount)
	for i := 0; i < count; i++ {
		offset, err := r.readInt32()
		if err != nil {
			return err
		}

		idx.Objects[i].Offset = uint64(offset)
	}

	return nil
}

func (r *Reader) read64bitsOffsets(idx *Idx) error {
	count := int(idx.ObjectCount)
	for i := 0; i < count; i++ {
		offset, err := r.readInt64()
		if err != nil {
			return err
		}

		if offset != 0 {
			idx.Objects[i].Offset = offset
		}

		fmt.Println(uint64(offset))
	}

	return nil
}

func (r *Reader) readChecksums(idx *Idx) error {
	if _, err := r.r.Read(idx.PackfileChecksum[:]); err != nil {
		return err
	}

	if _, err := r.r.Read(idx.IdxChecksum[:]); err != nil {
		return err
	}

	return nil
}

func (r *Reader) isValid(idx *Idx) bool {
	fanout := calculateFanout(idx)
	for k, c := range idx.Fanout {
		if fanout[k] != c {
			return false
		}
	}

	return true
}

func (r *Reader) readInt32() (uint32, error) {
	var value uint32
	if err := binary.Read(r.r, binary.BigEndian, &value); err != nil {
		return 0, err
	}

	return value, nil
}

func (r *Reader) readInt64() (uint64, error) {
	var value uint64
	if err := binary.Read(r.r, binary.BigEndian, &value); err != nil {
		return 0, err
	}

	return value, nil
}

func calculateFanout(idx *Idx) [255]uint32 {
	fanout := [255]uint32{}
	var c uint32
	for _, e := range idx.Objects {
		c++
		fanout[e.Hash[0]] = c
	}

	var i uint32
	for k, c := range fanout {
		if c != 0 {
			i = c
		}

		fanout[k] = i
	}

	return fanout
}
