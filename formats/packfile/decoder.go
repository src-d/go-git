package packfile

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/packfile/internal/readcounter"
	"gopkg.in/src-d/go-git.v3/storage/memory"
)

// Format specifies if the packfile uses ref-deltas or ofs-deltas.
type Format int

// Possible values of the Format type.
const (
	UnknownFormat  Format = 0
	OFSDeltaFormat Format = 1
	REFDeltaFormat Format = 2
)

var (
	// ErrEmptyPackfile is returned by Decode when no data is found in the packfile
	ErrEmptyPackfile = newDecoderError("empty packfile")
	// ErrUnsupportedVersion is returned by Decode when packfile version is different than VersionSupported.
	ErrUnsupportedVersion = newDecoderError("unsupported packfile version")
	// ErrMaxObjectsLimitReached is returned by Decode when the number of objects in the packfile is higher than Decoder.MaxObjectsLimit.
	ErrMaxObjectsLimitReached = newDecoderError("max. objects limit reached")
	// ErrMalformedPackfile is returned by Decode when the packfile is corrupt.
	ErrMalformedPackfile = newDecoderError("malformed pack file, does not start with 'PACK'")
	// ErrInvalidObject is returned by Decode when an invalid object is found in the packfile.
	ErrInvalidObject = newDecoderError("invalid git object")
	// ErrPackEntryNotFound is returned by Decode when a reference in the packfile references and unknown object.
	ErrPackEntryNotFound = newDecoderError("can't find a pack entry")
	// ErrZLib is returned by Decode when there was an error unzipping the packfile contents.
	ErrZLib = newDecoderError("zlib reading error")
)

const (
	// DefaultMaxObjectsLimit is the maximum amount of objects the decoder will decode before
	// returning ErrMaxObjectsLimitReached.
	DefaultMaxObjectsLimit = 1 << 20
	// VersionSupported is the packfile version supported by this decoder.
	VersionSupported = 2
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

	readCounter *readcounter.ReadCounter
	s           core.ObjectStorage
	offsets     map[int64]core.Hash
}

// NewDecoder returns a new Decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		MaxObjectsLimit: DefaultMaxObjectsLimit,

		readCounter: readcounter.New(r),
		offsets:     make(map[int64]core.Hash, 0),
	}
}

// Decode reads a packfile and stores it in the value pointed to by s.
func (d *Decoder) Decode(s core.ObjectStorage) (int64, error) {
	d.s = s
	if err := d.validateHeader(); err != nil {
		if err == io.EOF {
			return -1, ErrEmptyPackfile
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

	return d.readCounter.Count(), d.readObjects(count)
}

func (d *Decoder) validateHeader() error {
	var header = make([]byte, 4)
	if _, err := io.ReadFull(d.readCounter, header); err != nil {
		return err
	}

	if !bytes.Equal(header, []byte{'P', 'A', 'C', 'K'}) {
		return ErrMalformedPackfile
	}

	return nil
}

func (d *Decoder) readInt32() (uint32, error) {
	var value uint32
	if err := binary.Read(d.readCounter, binary.BigEndian, &value); err != nil {
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
		start := d.readCounter.Count()
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

	objectStart := d.readCounter.Count()

	typ, length, err := readTypeAndLength(d.readCounter)
	if err != nil {
		return nil, err
	}

	switch typ {
	case core.REFDeltaObject:
		content, typ, err = readContentREFDelta(d.readCounter, d)
		length = int64(len(content))
	case core.OFSDeltaObject:
		content, typ, err = readContentOFSDelta(d.readCounter, objectStart, d)
		length = int64(len(content))
	case core.CommitObject, core.TreeObject, core.BlobObject, core.TagObject:
		content, err = readContent(d.readCounter)
	default:
		err = ErrInvalidObject.addDetails("tag %q", typ)
	}
	if err != nil {
		return nil, err
	}

	return memory.NewObject(typ, length, content), err
}

// ByHash returns an already seen object by its hash.
func (d *Decoder) ByHash(hash core.Hash) (core.Object, error) {
	return d.s.Get(hash)
}

// ByOffset returns an already seen object by its offset in the packfile.
func (d *Decoder) ByOffset(offset int64) (core.Object, error) {
	hash, ok := d.offsets[offset]
	if !ok {
		return nil, ErrPackEntryNotFound.addDetails("offset %d", offset)
	}

	return d.ByHash(hash)
}

// DecoderError specifies errors returned by Decode.
type DecoderError struct {
	reason, details string
}

func newDecoderError(reason string) *DecoderError {
	return &DecoderError{reason: reason}
}

func (e *DecoderError) Error() string {
	if e.details == "" {
		return e.reason
	}

	return fmt.Sprintf("%s: %s", e.reason, e.details)
}

func (e *DecoderError) addDetails(format string, args ...interface{}) *DecoderError {
	return &DecoderError{
		reason:  e.reason,
		details: fmt.Sprintf(format, args...),
	}
}
