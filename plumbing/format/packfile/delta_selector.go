package packfile

import (
	"sort"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

const (
	// deltas based on deltas, how much steps we can do
	maxDepth = int64(50)
)

// set of object types that we should apply deltas
var applyDelta = map[plumbing.ObjectType]bool{
	plumbing.BlobObject: true,
	plumbing.TreeObject: true,
}

type deltaSelector struct {
	storer storer.EncodedObjectStorer
	depths map[plumbing.Hash]int
}

func newDeltaSelector(s storer.EncodedObjectStorer) *deltaSelector {
	return &deltaSelector{
		storer: s,
	}
}

func (dw *deltaSelector) getObjectsToPack(hashes []plumbing.Hash) ([]*ObjectToPack, error) {
	var objectsToPack []*ObjectToPack
	for _, h := range hashes {
		o, err := dw.storer.EncodedObject(plumbing.AnyObject, h)
		if err != nil {
			return nil, err
		}

		objectsToPack = append(objectsToPack, newObjectToPack(o))
	}

	return objectsToPack, nil
}

// GetObjectsToPack creates a list of ObjectToPack from the hashes provided,
// creating deltas if it's suitable, using an specific internal logic
func (dw *deltaSelector) GetObjectsToPack(hashes []plumbing.Hash) ([]*ObjectToPack, error) {
	otp, err := dw.getObjectsToPack(hashes)
	if err != nil {
		return nil, err
	}

	dw.sort(otp)

	if err := dw.walk(otp); err != nil {
		return nil, err
	}

	return otp, nil
}

func (dw *deltaSelector) sort(objectsToPack []*ObjectToPack) {
	sort.Sort(byTypeAndSize(objectsToPack))
}

func (dw *deltaSelector) walk(objectsToPack []*ObjectToPack) error {
	for i := 0; i < len(objectsToPack); i++ {
		target := objectsToPack[i]

		// We only want to create deltas from specific types
		if !applyDelta[target.Original.Type()] {
			continue
		}

		for j := i - 1; j > 0; j-- {
			base := objectsToPack[j]
			// Objects must use only the same type as their delta base.
			if base.Original.Type() != target.Original.Type() {
				break
			}

			delta, err := dw.tryToDeltify(base, target)
			if err != nil {
				return err
			}

			if delta != nil {
				objectsToPack[i] = delta
			}
		}
	}

	return nil
}

func (dw *deltaSelector) tryToDeltify(
	base, target *ObjectToPack) (*ObjectToPack, error) {

	// If the sizes are radically different, this is a bad pairing.
	if target.Original.Size() < base.Original.Size()>>4 {
		return nil, nil
	}

	msz := dw.deltaSizeLimit(base, target)
	// Nearly impossible to fit useful delta.
	if msz <= 8 {
		return nil, nil
	}

	// If we have to insert a lot to make this work, find another.
	if target.Original.Size()-base.Original.Size() > msz {
		return nil, nil
	}

	// Now we can generate the delta using originals
	delta, err := GetOFSDelta(base.Original, target.Original)
	if err != nil {
		return nil, err
	}

	deltaToPack := newDeltaObjectToPack(base, target.Original, delta)

	// if delta better than target
	if target.IsDelta() {
		newMsz := dw.deltaSizeLimit(base, deltaToPack)
		if newMsz > msz {
			return target, nil
		}
	}

	return deltaToPack, nil
}

func (dw *deltaSelector) deltaSizeLimit(base, target *ObjectToPack) int64 {
	if !target.IsDelta() {
		// Any delta should be no more than 50% of the original size
		// (for text files deflate of whole form should shrink 50%).
		n := target.Object.Size() >> 1

		// Evenly distribute delta size limits over allowed depth.
		// If src is non-delta (depth = 0), delta <= 50% of original.
		// If src is almost at limit (9/10), delta <= 10% of original.
		return n * (maxDepth - int64(base.Depth)) / maxDepth
	}

	// With a delta base chosen any new delta must be "better".
	// Retain the distribution described above.
	d := int64(target.Depth)
	n := target.Original.Size()

	// If src is whole (depth=0) and base is near limit (depth=9/10)
	// any delta using src can be 10x larger and still be better.
	//
	// If src is near limit (depth=9/10) and base is whole (depth=0)
	// a new delta dependent on src must be 1/10th the size.
	return n * (maxDepth - int64(base.Depth)) / (maxDepth - d)
}

type byTypeAndSize []*ObjectToPack

func (a byTypeAndSize) Len() int { return len(a) }

func (a byTypeAndSize) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a byTypeAndSize) Less(i, j int) bool {
	if a[i].Object.Type() < a[j].Object.Type() {
		return false
	}

	if a[i].Object.Type() > a[j].Object.Type() {
		return true
	}

	return a[i].Object.Size() > a[j].Object.Size()
}
