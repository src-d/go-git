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
// pack file. If it is a delta, Base is the object that this delta is based on
// (it could be also a delta). Original is the object that we can generate
// applying the delta to Base, or the same object as Object in the case of a
// non-delta object.
type ObjectToPack struct {
	Object   Object
	Base     *ObjectToPack
	Original Object
	Depth    int
}

// NewObjectToPack creates a correct ObjectToPack based on a non-delta object
func NewObjectToPack(o Object) *ObjectToPack {
	return &ObjectToPack{
		Object:   o,
		Original: o,
	}
}

// NewDeltaObjectToPack creates a correct ObjectToPack for a delta object, based on
// his base (could be another delta), the delta target (in this case called original),
// and the delta Object itself
func NewDeltaObjectToPack(base *ObjectToPack, original, delta Object) *ObjectToPack {
	return &ObjectToPack{
		Object:   delta,
		Base:     base,
		Original: original,
		Depth:    base.Depth + 1,
	}
}

func (o *ObjectToPack) IsDelta() bool {
	if o.Base != nil {
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
