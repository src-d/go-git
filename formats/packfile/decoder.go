package packfile

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/storage/memory"
)

type Format int

var (
	ErrEmptyRepository        = newError("empty repository")
	ErrUnsupportedVersion     = newError("unsupported packfile version")
	ErrMaxObjectsLimitReached = newError("max. objects limit reached")
	ErrMalformedPackfile      = newError("malformed pack file, does not start with 'PACK'")
	ErrInvalidObject          = newError("invalid git object")
	ErrPatching               = newError("patching error")
	ErrPackEntryNotFound      = newError("can't find a pack entry")
	ErrObjectNotFound         = newError("can't find a object")
	ErrZLib                   = newError("zlib reading error")
)

const (
	DefaultMaxObjectsLimit        = 1 << 20
	VersionSupported              = 2
	UnknownFormat          Format = 0
	OFSDeltaFormat         Format = 1
	REFDeltaFormat         Format = 2
)

// Decoder reads and decodes packfiles from an input stream.
type Decoder struct {
	// MaxObjectsLimit is the limit of objects to be load in the packfile, if
	// a packfile excess this number an error is throw, the default value
	// is defined by DefaultMaxObjectsLimit, usually the default limit is more
	// than enough to work with any repository, with higher values and huge
	// repositories you can run out of memory.
	MaxObjectsLimit uint32

	// Format specifies if we are using ref-delta's or ofs-delta's, by choosing the
	// correct format the memory usage is optimized
	// https://github.com/git/git/blob/8d530c4d64ffcc853889f7b385f554d53db375ed/Documentation/technical/protocol-capabilities.txt#L154
	Format Format

	r       *trackingReader
	s       core.ObjectStorage
	offsets map[int64]core.Hash
}

// NewDecoder returns a new Decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		MaxObjectsLimit: DefaultMaxObjectsLimit,

		r:       NewTrackingReader(r),
		offsets: make(map[int64]core.Hash, 0),
	}
}

// Decode reads a packfile and stores it in the value pointed to by s.
func (d *Decoder) Decode(s core.ObjectStorage) (int64, error) {
	d.s = s
	if err := d.validateHeader(); err != nil {
		if err == io.EOF {
			return -1, ErrEmptyRepository
		}

		return -1, err
	}

	version, err := d.readInt32()
	if err != nil {
		return -1, err
	}

	if version > VersionSupported {
		return -1, ErrUnsupportedVersion
	}

	count, err := d.readInt32()
	if err != nil {
		return -1, err
	}

	if count > d.MaxObjectsLimit {
		return -1, ErrMaxObjectsLimitReached
	}

	return d.r.position, d.readObjects(count)
}

func (d *Decoder) validateHeader() error {
	var header = make([]byte, 4)
	if _, err := io.ReadFull(d.r, header); err != nil {
		return err
	}

	if !bytes.Equal(header, []byte{'P', 'A', 'C', 'K'}) {
		return ErrMalformedPackfile
	}

	return nil
}

func (d *Decoder) readInt32() (uint32, error) {
	var value uint32
	if err := binary.Read(d.r, binary.BigEndian, &value); err != nil {
		return 0, err
	}

	return value, nil
}

func (d *Decoder) readObjects(count uint32) error {
	// This code has 50-80 µs of overhead per object not counting zlib inflation.
	// Together with zlib inflation, it's 400-410 µs for small objects.
	// That's 1 sec for ~2450 objects, ~4.20 MB, or ~250 ms per MB,
	// of which 12-20 % is _not_ zlib inflation (ie. is our code).
	for i := 0; i < int(count); i++ {
		start := d.r.position
		obj, err := d.newObject()
		if err != nil && err != io.EOF {
			return err
		}

		if d.Format == UnknownFormat || d.Format == OFSDeltaFormat {
			d.offsets[start] = obj.Hash()
		}

		_, err = d.s.Set(obj)
		if err == io.EOF {
			break
		}
	}

	return nil
}

func (d *Decoder) newObject() (core.Object, error) {
	var typ core.ObjectType
	var length int64
	var content []byte

	objectStart := d.r.position

	typ, length, err := readTypeAndLength(d.r)
	if err != nil {
		return nil, err
	}

	switch typ {
	case core.REFDeltaObject:
		content, typ, err = readContentREFDelta(d.r, d)
		length = int64(len(content))
	case core.OFSDeltaObject:
		content, typ, err = readContentOFSDelta(d.r, objectStart, d)
		length = int64(len(content))
	case core.CommitObject, core.TreeObject, core.BlobObject, core.TagObject:
		content, err = readContent(d.r)
	default:
		err = ErrInvalidObject.n("tag %q", typ)
	}
	if err != nil {
		return nil, err
	}

	return memory.NewObject(typ, length, content), err
}

// Returns an already seen object by its hash, part of Rememberer interface.
func (d *Decoder) ByHash(hash core.Hash) (core.Object, error) {
	return d.s.Get(hash)
}

// Returns an already seen object by its offset in the packfile, part of Rememberer interface.
func (d *Decoder) ByOffset(offset int64) (core.Object, error) {
	hash, ok := d.offsets[offset]
	if !ok {
		return nil, ErrPackEntryNotFound.n("offset %d", offset)
	}

	return d.ByHash(hash)
}

type ReaderError struct {
	reason, additional string
}

func newError(reason string) *ReaderError {
	return &ReaderError{reason: reason}
}

func (e *ReaderError) Error() string {
	if e.additional == "" {
		return e.reason
	}

	return fmt.Sprintf("%s: %s", e.reason, e.additional)
}

func (e *ReaderError) n(format string, args ...interface{}) *ReaderError {
	return &ReaderError{
		reason:     e.reason,
		additional: fmt.Sprintf(format, args...),
	}
}
