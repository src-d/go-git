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
	EmptyRepositoryErr        = newError("empty repository")
	UnsupportedVersionErr     = newError("unsupported packfile version")
	MaxObjectsLimitReachedErr = newError("max. objects limit reached")
	MalformedPackfileErr      = newError("malformed pack file, does not start with 'PACK'")
	InvalidObjectErr          = newError("invalid git object")
	PatchingErr               = newError("patching error")
	PackEntryNotFoundErr      = newError("can't find a pack entry")
	ErrObjectNotFound         = newError("can't find a object")
	ZLibErr                   = newError("zlib reading error")
)

const (
	DefaultMaxObjectsLimit = 1 << 20

	VersionSupported        = 2
	UnknownFormat    Format = 0
	OFSDeltaFormat   Format = 1
	REFDeltaFormat   Format = 2
)

// Reader reads a packfile from a binary string splitting it on objects
type Reader struct {
	// MaxObjectsLimit is the limit of objects to be load in the packfile, if
	// a packfile excess this number an error is throw, the default value
	// is defined by DefaultMaxObjectsLimit, usually the default limit is more
	// than enough to work with any repository, working extremly big repositories
	// where the number of object is bigger the memory can be exhausted.
	MaxObjectsLimit uint32

	// Format specifies if we are using ref-delta's or ofs-delta's, choosing the
	// correct format the memory usage is optimized
	// https://github.com/git/git/blob/8d530c4d64ffcc853889f7b385f554d53db375ed/Documentation/technical/protocol-capabilities.txt#L154
	Format Format

	r       *trackingReader
	s       core.ObjectStorage
	offsets map[int64]core.Hash
}

// NewReader returns a new Reader that reads from a io.Reader
func NewReader(r io.Reader) *Reader {
	return &Reader{
		MaxObjectsLimit: DefaultMaxObjectsLimit,

		r:       NewTrackingReader(r),
		offsets: make(map[int64]core.Hash, 0),
	}
}

// Read reads the objects and stores it at the ObjectStorage
func (r *Reader) Read(s core.ObjectStorage) (int64, error) {
	r.s = s
	if err := r.validateHeader(); err != nil {
		if err == io.EOF {
			return -1, EmptyRepositoryErr
		}

		return -1, err
	}

	version, err := r.readInt32()
	if err != nil {
		return -1, err
	}

	if version > VersionSupported {
		return -1, UnsupportedVersionErr
	}

	count, err := r.readInt32()
	if err != nil {
		return -1, err
	}

	if count > r.MaxObjectsLimit {
		return -1, MaxObjectsLimitReachedErr
	}

	return r.r.position, r.readObjects(count)
}

func (r *Reader) validateHeader() error {
	var header = make([]byte, 4)
	if _, err := io.ReadFull(r.r, header); err != nil {
		return err
	}

	if !bytes.Equal(header, []byte{'P', 'A', 'C', 'K'}) {
		return MalformedPackfileErr
	}

	return nil
}

func (r *Reader) readInt32() (uint32, error) {
	var value uint32
	if err := binary.Read(r.r, binary.BigEndian, &value); err != nil {
		return 0, err
	}

	return value, nil
}

func (r *Reader) readObjects(count uint32) error {
	// This code has 50-80 µs of overhead per object not counting zlib inflation.
	// Together with zlib inflation, it's 400-410 µs for small objects.
	// That's 1 sec for ~2450 objects, ~4.20 MB, or ~250 ms per MB,
	// of which 12-20 % is _not_ zlib inflation (ie. is our code).
	for i := 0; i < int(count); i++ {
		start := r.r.position
		obj, err := r.newObject()
		if err != nil && err != io.EOF {
			return err
		}

		if r.Format == UnknownFormat || r.Format == OFSDeltaFormat {
			r.offsets[start] = obj.Hash()
		}

		_, err = r.s.Set(obj)
		if err == io.EOF {
			break
		}
	}

	return nil
}

func (r *Reader) newObject() (core.Object, error) {
	var typ core.ObjectType
	var length int64
	var content []byte

	objectStart := r.r.position

	typ, length, err := readTypeAndLength(r.r)
	if err != nil {
		return nil, err
	}

	switch typ {
	case core.REFDeltaObject:
		content, typ, err = readContentREFDelta(r.r, r)
		length = int64(len(content))
	case core.OFSDeltaObject:
		content, typ, err = readContentOFSDelta(r.r, objectStart, r)
		length = int64(len(content))
	case core.CommitObject, core.TreeObject, core.BlobObject, core.TagObject:
		content, err = readContent(r.r)
	default:
		err = InvalidObjectErr.n("tag %q", typ)
	}
	if err != nil {
		return nil, err
	}

	return memory.NewObject(typ, length, content), err
}

// Returns an already seen object by its hash, part of Rememberer interface.
func (r *Reader) ByHash(hash core.Hash) (core.Object, error) {
	return r.s.Get(hash)
}

// Returns an already seen object by its offset in the packfile, part of Rememberer interface.
func (r *Reader) ByOffset(offset int64) (core.Object, error) {
	hash, ok := r.offsets[offset]
	if !ok {
		return nil, PackEntryNotFoundErr.n("offset %d", offset)
	}

	return r.ByHash(hash)
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
