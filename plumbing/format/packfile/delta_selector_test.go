package packfile

import (
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/memory"

	. "gopkg.in/check.v1"
)

type DeltaSelectorSuite struct {
	ds     *deltaSelector
	store  *memory.Storage
	hashes map[string]plumbing.Hash
}

var _ = Suite(&DeltaSelectorSuite{})

func (s *DeltaSelectorSuite) SetUpTest(c *C) {
	s.store = memory.NewStorage()
	s.createTestObjects()
	s.ds = newDeltaSelector(s.store)
}

func (s *DeltaSelectorSuite) TestSort(c *C) {
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

type testObject struct {
	id     string
	object plumbing.EncodedObject
}

var testObjects []*testObject = []*testObject{{
	id:     "base",
	object: newObject(plumbing.BlobObject, []byte("hello cruel world test")),
}, {
	id:     "o1",
	object: newObject(plumbing.BlobObject, []byte("1111111111111111111111111111")),
}, {
	id: "o2",
	object: newObject(plumbing.BlobObject, []byte("1111111111111111111111111111 "+
		"222222222222222222222222222222")),
}, {
	id: "o3",
	object: newObject(plumbing.BlobObject, []byte("1111111111111111111111111111 "+
		"222222222222222222222222222222 3333333333333333333333333333333333")),
}, {
	id: "bigBase",
	object: newObject(plumbing.BlobObject,
		genBytes([]piece{{
			times: 100000,
			val:   "a",
		}})),
}, {
	id:     "target",
	object: newObject(plumbing.BlobObject, []byte("hello world")),
}, {
	id: "changedTarget",
	object: newObject(plumbing.BlobObject,
		genBytes([]piece{{
			times: 1000,
			val:   "a",
		}})),
}, {
	id:     "goodTarget",
	object: newObject(plumbing.BlobObject, []byte("hello goat test test test")),
}, {
	id: "treeType",
	object: newObject(plumbing.TreeObject,
		[]byte("I am a tree!")),
}}

func (s *DeltaSelectorSuite) createTestObjects() {
	s.hashes = make(map[string]plumbing.Hash)
	for _, o := range testObjects {
		h, err := s.store.SetEncodedObject(o.object)
		if err != nil {
			panic(err)
		}
		s.hashes[o.id] = h
	}
}

func (s *DeltaSelectorSuite) TestObjectsToPack(c *C) {
	// Different type
	hashes := []plumbing.Hash{s.hashes["base"], s.hashes["treeType"]}
	otp, err := s.ds.ObjectsToPack(hashes)
	c.Assert(err, IsNil)
	c.Assert(len(otp), Equals, 2)
	c.Assert(otp[0].Object, Equals, s.store.Objects[s.hashes["base"]])
	c.Assert(otp[1].Object, Equals, s.store.Objects[s.hashes["treeType"]])

	// Size radically different
	hashes = []plumbing.Hash{s.hashes["bigBase"], s.hashes["target"]}
	otp, err = s.ds.ObjectsToPack(hashes)
	c.Assert(err, IsNil)
	c.Assert(len(otp), Equals, 2)
	c.Assert(otp[0].Object, Equals, s.store.Objects[s.hashes["bigBase"]])
	c.Assert(otp[1].Object, Equals, s.store.Objects[s.hashes["target"]])

	// Delta Size Limit with no best delta yet
	hashes = []plumbing.Hash{s.hashes["base"], s.hashes["target"]}
	otp, err = s.ds.ObjectsToPack(hashes)
	c.Assert(err, IsNil)
	c.Assert(len(otp), Equals, 2)
	c.Assert(otp[0].Object, Equals, s.store.Objects[s.hashes["base"]])
	c.Assert(otp[1].Object, Equals, s.store.Objects[s.hashes["target"]])

	// They need to insert a lot to create the delta
	hashes = []plumbing.Hash{s.hashes["base"], s.hashes["changedTarget"]}
	otp, err = s.ds.ObjectsToPack(hashes)
	c.Assert(err, IsNil)
	c.Assert(len(otp), Equals, 2)
	c.Assert(otp[0].Object, Equals, s.store.Objects[s.hashes["changedTarget"]])
	c.Assert(otp[1].Object, Equals, s.store.Objects[s.hashes["base"]])

	// It will create the delta
	hashes = []plumbing.Hash{s.hashes["o1"], s.hashes["o2"]}
	otp, err = s.ds.ObjectsToPack(hashes)
	c.Assert(err, IsNil)
	c.Assert(len(otp), Equals, 2)
	c.Assert(otp[0].Object, Equals, s.store.Objects[s.hashes["o2"]])
	c.Assert(otp[0].IsDelta(), Equals, false)
	c.Assert(otp[1].Original, Equals, s.store.Objects[s.hashes["o1"]])
	c.Assert(otp[1].IsDelta(), Equals, true)
	c.Assert(otp[1].Depth, Equals, 1)

	// If our base is another delta, the depth will increase by one
	hashes = []plumbing.Hash{s.hashes["o1"], s.hashes["o2"], s.hashes["o3"]}
	otp, err = s.ds.ObjectsToPack(hashes)
	c.Assert(err, IsNil)
	c.Assert(len(otp), Equals, 3)
	c.Assert(otp[0].Object, Equals, s.store.Objects[s.hashes["o3"]])
	c.Assert(otp[0].IsDelta(), Equals, false)
	c.Assert(otp[1].Original, Equals, s.store.Objects[s.hashes["o2"]])
	c.Assert(otp[1].IsDelta(), Equals, true)
	c.Assert(otp[1].Depth, Equals, 1)
	c.Assert(otp[2].Original, Equals, s.store.Objects[s.hashes["o1"]])
	c.Assert(otp[2].IsDelta(), Equals, true)
	c.Assert(otp[2].Depth, Equals, 1)
}
