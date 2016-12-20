package storer

// Storer is a full storer for encoded objects, references, index and shallow.
type Storer interface {
	EncodedObjectStorer
	IndexStorer
	ReferenceStorer
	ShallowStorer
}
