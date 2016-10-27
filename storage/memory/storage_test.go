package memory_test

import (
	"testing"

	"gopkg.in/src-d/go-git.v4/storage/test"
	"gopkg.in/src-d/go-git.v4/storage/memory"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type StorageSuite struct {
	test.BaseStorageSuite
}

var _ = Suite(&StorageSuite{})

func (s *StorageSuite) SetUpTest(c *C) {
	s.BaseStorageSuite = test.NewBaseStorageSuite(memory.NewStorage())
}

func (s *StorageSuite) TearDownTest(c *C) {
	c.Assert(s.BaseStorageSuite.Storage.Close(), IsNil)
}

func (s *StorageSuite) TestStorageObjectStorage(c *C) {
	storage := memory.NewStorage()
	o := storage.ObjectStorage()
	e := storage.ObjectStorage()

	c.Assert(o == e, Equals, true)
}

func (s *StorageSuite) TestStorageReferenceStorage(c *C) {
	storage := memory.NewStorage()
	o := storage.ReferenceStorage()
	e := storage.ReferenceStorage()

	c.Assert(o == e, Equals, true)
}

func (s *StorageSuite) TestStorageConfigStorage(c *C) {
	storage := memory.NewStorage()
	o := storage.ConfigStorage()
	e := storage.ConfigStorage()

	c.Assert(o == e, Equals, true)
}
