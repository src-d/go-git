package packfile

import (
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/memory"

	. "gopkg.in/check.v1"
)

type DeltaSelectorSuite struct {
	ds    *deltaSelector
	store *memory.Storage
}

var _ = Suite(&DeltaSelectorSuite{})

func (s *DeltaSelectorSuite) SetUpTest(c *C) {
	s.store = memory.NewStorage()
	s.ds = newDeltaSelector(s.store)
}

func (s *DeltaSelectorSuite) TestTryOrder(c *C) {
	var o1 = newObjectToPack(newObject(plumbing.BlobObject, []byte("00000")))
	var o4 = newObjectToPack(newObject(plumbing.BlobObject, []byte("0000")))
	var o6 = newObjectToPack(newObject(plumbing.BlobObject, []byte("00")))
	var o9 = newObjectToPack(newObject(plumbing.BlobObject, []byte("0")))
	var o8 = newObjectToPack(newObject(plumbing.TreeObject, []byte("000")))
	var o2 = newObjectToPack(newObject(plumbing.TreeObject, []byte("00")))
	var o3 = newObjectToPack(newObject(plumbing.TreeObject, []byte("0")))
	var o5 = newObjectToPack(newObject(plumbing.CommitObject, []byte("0000")))
	var o7 = newObjectToPack(newObject(plumbing.CommitObject, []byte("00")))

	toSort := []*ObjectToPack{o1, o2, o3, o4, o5, o6, o7, o8, o9}
	s.ds.sort(toSort)
	expected := []*ObjectToPack{o1, o4, o6, o9, o8, o2, o3, o5, o7}
	c.Assert(toSort, DeepEquals, expected)
}

func (s *DeltaSelectorSuite) TestTryToDeltify(c *C) {
	base := newObjectToPack(newObject(plumbing.BlobObject,
		[]byte("hello cruel world")))

	// Size radically different
	bigBase := newObjectToPack(newObject(plumbing.BlobObject,
		genBytes([]piece{{
			times: 100000,
			val:   "a",
		}})))
	target := newObjectToPack(newObject(plumbing.BlobObject,
		[]byte("hello world")))
	bestDelta, err := s.ds.tryToDeltify(bigBase, target)
	c.Assert(err, IsNil)
	c.Assert(bestDelta, IsNil)

	// Delta Size Limit with no best delta yet
	bestDelta, err = s.ds.tryToDeltify(base, target)
	c.Assert(err, IsNil)
	//c.Assert(bestDelta, IsNil)

	// They need to insert a lot to create the delta
	target = newObjectToPack(newObject(plumbing.BlobObject,
		[]byte("hello cruel world aaaaaaaaaa bbbbbbb")))
	bestDelta, err = s.ds.tryToDeltify(base, target)
	c.Assert(err, IsNil)
	//c.Assert(bestDelta, IsNil)

	// It will create the delta
	target = newObjectToPack(newObject(plumbing.BlobObject,
		[]byte("hello cruel world test")))
	bestDelta, err = s.ds.tryToDeltify(base, target)
	c.Assert(err, IsNil)
	c.Assert(bestDelta, NotNil)
	c.Assert(bestDelta.Original, DeepEquals, target.Original)
	c.Assert(bestDelta.Base, DeepEquals, base)
	c.Assert(bestDelta.Depth, Equals, 1)

	// If our base is another delta, the depth will increase by one
	otherTarget := newObjectToPack(newObject(plumbing.BlobObject,
		[]byte("hello cruel world test test")))
	otherBestDelta, err := s.ds.tryToDeltify(bestDelta, otherTarget)
	c.Assert(err, IsNil)
	c.Assert(otherBestDelta, NotNil)
	c.Assert(otherBestDelta.Original, DeepEquals, otherTarget.Original)
	c.Assert(otherBestDelta.Depth, Equals, 2)
}
