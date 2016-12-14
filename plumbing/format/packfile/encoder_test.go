package packfile

import (
	"bytes"

	"gopkg.in/src-d/go-git.v4/fixtures"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/memory"

	. "gopkg.in/check.v1"
)

type EncoderSuite struct {
	fixtures.Suite
	buf   *bytes.Buffer
	store *memory.Storage
	enc   *Encoder
}

var _ = Suite(&EncoderSuite{})

func (s *EncoderSuite) SetUpTest(c *C) {
	s.buf = bytes.NewBuffer(nil)
	s.store = memory.NewStorage()
	s.enc = NewEncoder(s.buf, s.store)
}

func (s *EncoderSuite) TestCorrectPackHeader(c *C) {
	hash, err := s.enc.Encode([]plumbing.Hash{})
	c.Assert(err, IsNil)

	hb := [20]byte(hash)

	// PACK + VERSION + OBJECTS + HASH
	expectedResult := []byte{'P', 'A', 'C', 'K', 0, 0, 0, 2, 0, 0, 0, 0}
	expectedResult = append(expectedResult, hb[:]...)

	result := s.buf.Bytes()

	c.Assert(result, DeepEquals, expectedResult)
}

func (s *EncoderSuite) TestCorrectPackWithOneEmptyObject(c *C) {
	o := &plumbing.MemoryObject{}
	o.SetType(plumbing.CommitObject)
	o.SetSize(0)
	_, err := s.store.SetEncodedObject(o)
	c.Assert(err, IsNil)

	hash, err := s.enc.Encode([]plumbing.Hash{o.Hash()})
	c.Assert(err, IsNil)

	// PACK + VERSION(2) + OBJECT NUMBER(1)
	expectedResult := []byte{'P', 'A', 'C', 'K', 0, 0, 0, 2, 0, 0, 0, 1}
	// OBJECT HEADER(TYPE + SIZE)= 0001 0000
	expectedResult = append(expectedResult, []byte{16}...)

	// Zlib header
	expectedResult = append(expectedResult,
		[]byte{120, 156, 1, 0, 0, 255, 255, 0, 0, 0, 1}...)

	// + HASH
	hb := [20]byte(hash)
	expectedResult = append(expectedResult, hb[:]...)

	result := s.buf.Bytes()

	c.Assert(result, DeepEquals, expectedResult)
}

func (s *EncoderSuite) TestMaxObjectSize(c *C) {
	o := s.store.NewEncodedObject()
	o.SetSize(9223372036854775807)
	o.SetType(plumbing.CommitObject)
	_, err := s.store.SetEncodedObject(o)
	c.Assert(err, IsNil)
	hash, err := s.enc.Encode([]plumbing.Hash{o.Hash()})
	c.Assert(err, IsNil)
	c.Assert(hash.IsZero(), Not(Equals), true)
}

func (s *EncoderSuite) TestHashNotFound(c *C) {
	h, err := s.enc.Encode([]plumbing.Hash{plumbing.NewHash("BAD")})
	c.Assert(h, Equals, plumbing.ZeroHash)
	c.Assert(err, NotNil)
	c.Assert(err, Equals, plumbing.ErrObjectNotFound)
}

func (s *EncoderSuite) TestDecodeEncodeDecode(c *C) {
	fixtures.Basic().ByTag("packfile").Test(c, func(f *fixtures.Fixture) {
		scanner := NewScanner(f.Packfile())
		storage := memory.NewStorage()

		d, err := NewDecoder(scanner, storage)
		c.Assert(err, IsNil)

		ch, err := d.Decode()
		c.Assert(err, IsNil)
		c.Assert(ch, Equals, f.PackfileHash)

		objIter, err := d.o.IterEncodedObjects(plumbing.AnyObject)
		c.Assert(err, IsNil)

		objects := []plumbing.EncodedObject{}
		hashes := []plumbing.Hash{}
		err = objIter.ForEach(func(o plumbing.EncodedObject) error {
			objects = append(objects, o)
			hash, err := s.store.SetEncodedObject(o)
			c.Assert(err, IsNil)

			hashes = append(hashes, hash)

			return err

		})
		c.Assert(err, IsNil)
		_, err = s.enc.Encode(hashes)
		c.Assert(err, IsNil)

		scanner = NewScanner(s.buf)
		storage = memory.NewStorage()
		d, err = NewDecoder(scanner, storage)
		c.Assert(err, IsNil)
		_, err = d.Decode()
		c.Assert(err, IsNil)

		objIter, err = d.o.IterEncodedObjects(plumbing.AnyObject)
		c.Assert(err, IsNil)
		obtainedObjects := []plumbing.EncodedObject{}
		err = objIter.ForEach(func(o plumbing.EncodedObject) error {
			obtainedObjects = append(obtainedObjects, o)

			return nil
		})
		c.Assert(err, IsNil)
		c.Assert(len(obtainedObjects), Equals, len(objects))

		equals := 0
		for _, oo := range obtainedObjects {
			for _, o := range objects {
				if o.Hash() == oo.Hash() {
					equals++
				}
			}
		}

		c.Assert(len(obtainedObjects), Equals, equals)
	})
}

func (s *EncoderSuite) TestDecodeEncodeWithDeltaDecodeREF(c *C) {
	s.simpleDeltaTest(c, plumbing.REFDeltaObject)
}

func (s *EncoderSuite) TestDecodeEncodeWithDeltaDecodeOFS(c *C) {
	s.simpleDeltaTest(c, plumbing.OFSDeltaObject)
}

func (s *EncoderSuite) TestDecodeEncodeWithDeltasDecodeREF(c *C) {
	s.deltaOverDeltaTest(c, plumbing.REFDeltaObject)
}

func (s *EncoderSuite) TestDecodeEncodeWithDeltasDecodeOFS(c *C) {
	s.deltaOverDeltaTest(c, plumbing.OFSDeltaObject)
}

func (s *EncoderSuite) simpleDeltaTest(c *C, t plumbing.ObjectType) {
	srcObject := newObject(plumbing.BlobObject, []byte("0"))
	targetObject := newObject(plumbing.BlobObject, []byte("01"))

	deltaObject, err := delta(srcObject, targetObject, t)
	c.Assert(err, IsNil)

	srcToPack := newObjectToPack(srcObject)
	_, err = s.enc.encode([]*ObjectToPack{
		srcToPack,
		newDeltaObjectToPack(srcToPack, targetObject, deltaObject),
	})
	c.Assert(err, IsNil)

	scanner := NewScanner(s.buf)

	storage := memory.NewStorage()
	d, err := NewDecoder(scanner, storage)
	c.Assert(err, IsNil)

	_, err = d.Decode()
	c.Assert(err, IsNil)

	decSrc, err := storage.EncodedObject(srcObject.Type(), srcObject.Hash())
	c.Assert(err, IsNil)
	c.Assert(decSrc, DeepEquals, srcObject)

	decTarget, err := storage.EncodedObject(targetObject.Type(), targetObject.Hash())
	c.Assert(err, IsNil)
	c.Assert(decTarget, DeepEquals, targetObject)
}

func (s *EncoderSuite) deltaOverDeltaTest(c *C, t plumbing.ObjectType) {
	srcObject := newObject(plumbing.BlobObject, []byte("0"))
	targetObject := newObject(plumbing.BlobObject, []byte("01"))
	otherTargetObject := newObject(plumbing.BlobObject, []byte("011111"))

	deltaObject, err := delta(srcObject, targetObject, t)
	c.Assert(err, IsNil)
	c.Assert(deltaObject.Hash(), Not(Equals), plumbing.ZeroHash)

	otherDeltaObject, err := delta(targetObject, otherTargetObject, t)
	c.Assert(err, IsNil)
	c.Assert(otherDeltaObject.Hash(), Not(Equals), plumbing.ZeroHash)

	srcToPack := newObjectToPack(srcObject)
	targetToPack := newObjectToPack(targetObject)
	_, err = s.enc.encode([]*ObjectToPack{
		srcToPack,
		newDeltaObjectToPack(srcToPack, targetObject, deltaObject),
		newDeltaObjectToPack(targetToPack, otherTargetObject, otherDeltaObject),
	})
	c.Assert(err, IsNil)

	scanner := NewScanner(s.buf)
	storage := memory.NewStorage()
	d, err := NewDecoder(scanner, storage)
	c.Assert(err, IsNil)

	_, err = d.Decode()
	c.Assert(err, IsNil)

	decSrc, err := storage.EncodedObject(srcObject.Type(), srcObject.Hash())
	c.Assert(err, IsNil)
	c.Assert(decSrc, DeepEquals, srcObject)

	decTarget, err := storage.EncodedObject(targetObject.Type(), targetObject.Hash())
	c.Assert(err, IsNil)
	c.Assert(decTarget, DeepEquals, targetObject)

	decOtherTarget, err := storage.EncodedObject(otherTargetObject.Type(), otherTargetObject.Hash())
	c.Assert(err, IsNil)
	c.Assert(decOtherTarget, DeepEquals, otherTargetObject)
}

func delta(base, target plumbing.EncodedObject, t plumbing.ObjectType) (plumbing.EncodedObject, error) {
	switch t {
	case plumbing.OFSDeltaObject:
		return GetOFSDelta(base, target)
	case plumbing.REFDeltaObject:
		return GetRefDelta(base, target)
	default:
		panic("delta type not found")
	}
}

func newObject(t plumbing.ObjectType, cont []byte) plumbing.EncodedObject {
	o := plumbing.MemoryObject{}
	o.SetType(t)
	o.SetSize(int64(len(cont)))
	o.Write(cont)

	return &o
}
