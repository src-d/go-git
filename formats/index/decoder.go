package index

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"
)

var (
	// ErrUnsupportedVersion is returned by Decode when the idxindex file
	// version is not supported.
	ErrUnsupportedVersion = errors.New("Unsuported version")
	// ErrMalformedSignature is returned by Decode when the index header file is
	// malformed
	ErrMalformedSignature = errors.New("Malformed index signature file")

	indexSignature = []byte{'D', 'I', 'R', 'C'}
)

const (
	// IndexVersionSupported is the only index version supported.
	IndexVersionSupported = 2
)

type Decoder struct {
	r io.Reader
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Decode reads the whole idx object from its input and stores it in the
// value pointed to by idx.
func (d *Decoder) Decode(idx *Index) error {
	version, err := validateHeader(d.r)
	if err != nil {
		return err
	}

	idx.Version = version

	if err := binary.Read(d.r, binary.BigEndian, &idx.EntryCount); err != nil {
		return err
	}

	return d.readEntries(idx)
}

func (d *Decoder) readEntries(idx *Index) error {
	for i := 0; i < int(idx.EntryCount); i++ {
		e, err := d.readEntry(idx)
		if err != nil {
			return err
		}

		idx.Entries = append(idx.Entries, *e)
	}

	return nil
}

func (d *Decoder) readEntry(idx *Index) (*Entry, error) {
	e := &Entry{}

	var msec, mnsec, sec, nsec uint32

	flow := []interface{}{
		&msec, &mnsec,
		&sec, &nsec,
		&e.Dev,
		&e.Inode,
		&e.Mode,
		&e.UID,
		&e.GID,
		&e.Size,
		&e.Hash,
		&e.Flags,
	}

	if err := readBinary(d.r, flow); err != nil {
		return nil, err
	}

	e.CreatedAt = time.Unix(int64(msec), int64(mnsec))
	e.ModifiedAt = time.Unix(int64(sec), int64(nsec))

	if err := d.readEntryName(e); err != nil {
		return nil, err
	}

	// Index entries are padded out to the next 8 byte alignment
	// for historical reasons related to how C Git read the files.
	entrySize := 62 + len(e.Name)
	padLen := 8 - entrySize%8
	if _, err := io.CopyN(ioutil.Discard, d.r, int64(padLen)); err != nil {
		return nil, err
	}

	return e, nil
}

const (
	NAME_MASK = 0xfff
)

func (d *Decoder) readEntryName(e *Entry) error {
	pLen := e.Flags & NAME_MASK

	name := make([]byte, int64(pLen))
	if err := binary.Read(d.r, binary.BigEndian, &name); err != nil {
		return err
	}

	fmt.Println(len(name), name, string(name))
	e.Name = string(name)

	return nil
}

func validateHeader(r io.Reader) (version uint32, err error) {
	var h = make([]byte, 4)
	if _, err := r.Read(h); err != nil {
		return 0, err
	}

	if !bytes.Equal(h, indexSignature) {
		return 0, ErrMalformedSignature
	}

	if err := binary.Read(r, binary.BigEndian, &version); err != nil {
		return 0, err
	}

	if version != IndexVersionSupported {
		return 0, ErrUnsupportedVersion
	}

	return
}

func readBinary(r io.Reader, data []interface{}) error {
	for _, v := range data {
		err := binary.Read(r, binary.BigEndian, v)
		if err != nil {
			return err
		}
	}

	return nil
}
