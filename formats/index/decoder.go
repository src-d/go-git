package index

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"strconv"
	"time"

	"gopkg.in/src-d/go-git.v4/core"
)

var (
	// IndexVersionSupported is the range of supported index versions
	IndexVersionSupported = struct{ Min, Max uint32 }{Min: 2, Max: 4}

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

const (
	EntryExtended = 0x4000
	EntryValid    = 0x8000

	nameMask         = 0xfff
	intentToAddMask  = 1 << 13
	skipWorkTreeMask = 1 << 14
)

type Decoder struct {
	r         io.Reader
	lastEntry *Entry
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

		d.lastEntry = e
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

	if err := readBinary(d.r, flow...); err != nil {
		return nil, err
	}

	read := 62
	e.CreatedAt = time.Unix(int64(msec), int64(mnsec))
	e.ModifiedAt = time.Unix(int64(sec), int64(nsec))
	e.Stage = Stage(e.Flags>>12) & 0x3

	if e.Flags&EntryExtended != 0 {
		read += 2
		var extended uint16
		if err := readBinary(d.r, &extended); err != nil {
			return nil, err
		}

		e.IntentToAdd = extended&intentToAddMask != 0
		e.SkipWorktree = extended&skipWorkTreeMask != 0
	}

	if err := d.readEntryName(idx, e); err != nil {
		return nil, err

	}

	return e, d.padEntry(idx, e, read)
}

func (d *Decoder) readEntryName(idx *Index, e *Entry) error {
	var name string
	var err error

	switch idx.Version {
	case 2, 3:
		name, err = d.doReadEntryName(e)
	case 4:
		name, err = d.doReadEntryNameV4()
	}

	if err != nil {
		return err
	}

	e.Name = name
	return nil
}

func (d *Decoder) doReadEntryNameV4() (string, error) {
	l, err := readVariableWidthInt(d.r)
	if err != nil {
		return "", err
	}

	var base string
	if d.lastEntry != nil {
		base = d.lastEntry.Name[:len(d.lastEntry.Name)-int(l)]
	}

	name, err := readUntil(d.r, 0)
	if err != nil {
		return "", err
	}

	return base + string(name), nil
}

func (d *Decoder) doReadEntryName(e *Entry) (string, error) {
	pLen := e.Flags & nameMask

	name := make([]byte, int64(pLen))
	if err := binary.Read(d.r, binary.BigEndian, &name); err != nil {
		return "", err
	}

	return string(name), nil
}

// Index entries are padded out to the next 8 byte alignment
// for historical reasons related to how C Git read the files.
func (d *Decoder) padEntry(idx *Index, e *Entry, read int) error {
	if idx.Version == 4 {
		return nil
	}

	entrySize := read + len(e.Name)
	padLen := 8 - entrySize%8
	if _, err := io.CopyN(ioutil.Discard, d.r, int64(padLen)); err != nil {
		return err
	}

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

	return err
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
	case bytes.Equal(s, resolveUndoExtSignature):
		ru := &ResolveUndo{}
		rud := &ResolveUndoDecoder{&io.LimitedReader{R: d.r, N: int64(len)}}
		if err := rud.Decode(ru); err != nil {
			return err
		}

		idx.ResolveUndo = ru
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

	if version < IndexVersionSupported.Min || version > IndexVersionSupported.Max {
		return 0, ErrUnsupportedVersion
	}

	return
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

		if e == nil {
			continue
		}

		t.Entries = append(t.Entries, *e)
	}
}

func (d *TreeExtensionDecoder) readEntry() (*TreeEntry, error) {
	e := &TreeEntry{}

	path, err := readUntil(d.r, '\x00')
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

	// An entry can be in an invalidated state and is represented by having a
	// negative number in the entry_count field.
	if i == -1 {
		return nil, nil
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

type ResolveUndoDecoder struct {
	r io.Reader
}

func (d *ResolveUndoDecoder) Decode(ru *ResolveUndo) error {
	for {
		e, err := d.readEntry()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		ru.Entries = append(ru.Entries, *e)
	}
}

func (d *ResolveUndoDecoder) readEntry() (*ResolveUndoEntry, error) {
	e := &ResolveUndoEntry{
		Stages: make(map[Stage]core.Hash, 0),
	}

	path, err := readUntil(d.r, 0)
	if err != nil {
		return nil, err
	}

	e.Path = string(path)

	for i := 0; i < 3; i++ {
		if err := d.readStage(e, Stage(i+1)); err != nil {
			return nil, err
		}
	}

	for s := range e.Stages {
		var hash core.Hash
		if err := binary.Read(d.r, binary.BigEndian, hash[:]); err != nil {
			return nil, err
		}

		e.Stages[s] = hash
	}

	return e, nil
}

func (d *ResolveUndoDecoder) readStage(e *ResolveUndoEntry, s Stage) error {
	ascii, err := readUntil(d.r, 0)
	if err != nil {
		return err
	}

	stage, err := strconv.ParseInt(string(ascii), 8, 64)
	if err != nil {
		return err
	}

	if stage != 0 {
		e.Stages[s] = core.ZeroHash
	}

	return nil
}

func readBinary(r io.Reader, data ...interface{}) error {
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

//     dheader[pos] = ofs & 127;
//     while (ofs >>= 7)
//         dheader[--pos] = 128 | (--ofs & 127);
//
func readVariableWidthInt(r io.Reader) (int64, error) {
	var c byte
	if err := readBinary(r, &c); err != nil {
		return 0, err
	}

	var v = int64(c & maskLength)
	for moreBytesInLength(c) {
		v++
		if err := readBinary(r, &c); err != nil {
			return 0, err
		}

		v = (v << lengthBits) + int64(c&maskLength)
	}

	return v, nil
}

const (
	maskContinue = uint8(128) // 1000 000
	maskLength   = uint8(127) // 0111 1111
	lengthBits   = uint8(7)   // subsequent bytes has 7 bits to store the length
)

func moreBytesInLength(c byte) bool {
	return c&maskContinue > 0
}