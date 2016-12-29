package plumbing

// Revision represents a git revision
type Revision string

func (r Revision) String() string {
	return string(r)
}
