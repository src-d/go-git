// package plumbing implement the core interfaces and structs used by go-git
package plumbing

import (
	"errors"
	"io"
)

var (
	ErrObjectNotFound = errors.New("object not found")
	// ErrInvalidType is returned when an invalid object type is provided.
	ErrInvalidType = errors.New("invalid object type")
)

// ObjectToPack is a representation of an object that is going to be into a
// pack file. If it is a delta, Source is the delta source and Original is the
// delta target
type ObjectToPack struct {
	Object
	Source   Object
	Original Object
}

func NewObjectToPack(o Object) *ObjectToPack {
	return &ObjectToPack{
		Object: o,
	}
}

func NewDeltaObjectToPack(base, original, delta Object) *ObjectToPack {
	return &ObjectToPack{
		Object:   delta,
		Source:   base,
		Original: original,
	}
}

func (o *ObjectToPack) IsDelta() bool {
	if o.Source != nil && o.Original != nil {
		return true
	}

	return false
}

// Object is a generic representation of any git object
type Object interface {
	Hash() Hash
	Type() ObjectType
	SetType(ObjectType)
	Size() int64
	SetSize(int64)
	Reader() (io.ReadCloser, error)
	Writer() (io.WriteCloser, error)
}

// ObjectType internal object type
// Integer values from 0 to 7 map to those exposed by git.
// AnyObject is used to represent any from 0 to 7.
type ObjectType int8

const (
	InvalidObject ObjectType = 0
	CommitObject  ObjectType = 1
	TreeObject    ObjectType = 2
	BlobObject    ObjectType = 3
	TagObject     ObjectType = 4
	// 5 reserved for future expansion
	OFSDeltaObject ObjectType = 6
	REFDeltaObject ObjectType = 7

	AnyObject ObjectType = -127
)

func (t ObjectType) String() string {
	switch t {
	case CommitObject:
		return "commit"
	case TreeObject:
		return "tree"
	case BlobObject:
		return "blob"
	case TagObject:
		return "tag"
	case OFSDeltaObject:
		return "ofs-delta"
	case REFDeltaObject:
		return "ref-delta"
	case AnyObject:
		return "any"
	default:
		return "unknown"
	}
}

func (t ObjectType) Bytes() []byte {
	return []byte(t.String())
}

// Valid returns true if t is a valid ObjectType.
func (t ObjectType) Valid() bool {
	return t >= CommitObject && t <= REFDeltaObject
}

// ParseObjectType parses a string representation of ObjectType. It returns an
// error on parse failure.
func ParseObjectType(value string) (typ ObjectType, err error) {
	switch value {
	case "commit":
		typ = CommitObject
	case "tree":
		typ = TreeObject
	case "blob":
		typ = BlobObject
	case "tag":
		typ = TagObject
	case "ofs-delta":
		typ = OFSDeltaObject
	case "ref-delta":
		typ = REFDeltaObject
	default:
		err = ErrInvalidType
	}
	return
}
