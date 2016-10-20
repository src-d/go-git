package index

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"strconv"
	"time"
)

var (
	// ErrUnsupportedVersion is returned by Decode when the idxindex file
	// version is not supported.
	ErrUnsupportedVersion = errors.New("Unsuported version")
	// ErrMalformedSignature is returned by Decode when the index header file is
	// malformed
	ErrMalformedSignature = errors.New("Malformed index signature file")

	indexSignature          = []byte{'D', 'I', 'R', 'C'}
	treeExtSignature        = []byte{'T', 'R', 'E', 'E'}
	resolveUndoExtSignature = []byte{'R', 'E', 'U', 'C'}
)

type Stage int

const (
	// IndexVersionSupported is the only index version supported.
	IndexVersionSupported = 2

	// Merged is the default stage, fully merged
	Merged Stage = 1
	// AncestorMode is the base revision
	AncestorMode Stage = 1
	// OurMode is the first tree revision, ours
	OurMode Stage = 2
	// TheirMode is the second tree revision, theirs
	TheirMode Stage = 3

	nameMask = 0xfff
)

type Decoder struct {
	r io.Reader
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Decode reads the whole index object from its input and stores it in the
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

	if err := d.readEntries(idx); err != nil {
		return err
	}

	return d.readExtensions(idx)
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

	e.Stage = Stage(e.Flags>>12) & 0x3
	return e, nil
}

func (d *Decoder) readEntryName(e *Entry) error {
	pLen := e.Flags & nameMask

	name := make([]byte, int64(pLen))
	if err := binary.Read(d.r, binary.BigEndian, &name); err != nil {
		return err
	}

	e.Name = string(name)
	return nil
}

func (d *Decoder) readExtensions(idx *Index) error {
	var err error
	for {
		err = d.readExtension(idx)
		if err != nil {
			break
		}
	}

	if err == io.EOF {
		return nil
	}

	return nil
}

func (d *Decoder) readExtension(idx *Index) error {
	var s = make([]byte, 4)
	if _, err := d.r.Read(s); err != nil {
		return err
	}

	var len uint32
	if err := binary.Read(d.r, binary.BigEndian, &len); err != nil {
		return err
	}

	switch {
	case bytes.Equal(s, treeExtSignature):
		t := &Tree{}
		td := &TreeExtensionDecoder{&io.LimitedReader{R: d.r, N: int64(len)}}
		if err := td.Decode(t); err != nil {
			return err
		}

		idx.Cache = t
	}

	return nil
}

func validateHeader(r io.Reader) (version uint32, err error) {
	var s = make([]byte, 4)
	if _, err := r.Read(s); err != nil {
		return 0, err
	}

	if !bytes.Equal(s, indexSignature) {
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

func readUntil(r io.Reader, delim byte) ([]byte, error) {
	var buf [1]byte
	value := make([]byte, 0, 16)
	for {
		if _, err := r.Read(buf[:]); err != nil {
			if err == io.EOF {
				return nil, err
			}

			return nil, err
		}

		if buf[0] == delim {
			return value, nil
		}

		value = append(value, buf[0])
	}
}

type TreeExtensionDecoder struct {
	r io.Reader
}

func (d *TreeExtensionDecoder) Decode(t *Tree) error {
	for {
		e, err := d.readEntry()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		t.Entries = append(t.Entries, *e)
	}
}

func (d *TreeExtensionDecoder) readEntry() (*TreeEntry, error) {
	e := &TreeEntry{}

	path, err := readUntil(d.r, 0)
	if err != nil {
		return nil, err
	}

	e.Path = string(path)

	count, err := readUntil(d.r, ' ')
	if err != nil {
		return nil, err
	}

	i, err := strconv.Atoi(string(count))
	if err != nil {
		return nil, err
	}

	e.Entries = i

	trees, err := readUntil(d.r, 10)
	if err != nil {
		return nil, err
	}

	i, err = strconv.Atoi(string(trees))
	if err != nil {
		return nil, err
	}

	e.Trees = i

	if err := binary.Read(d.r, binary.BigEndian, &e.Hash); err != nil {
		return nil, err
	}

	return e, nil
}
