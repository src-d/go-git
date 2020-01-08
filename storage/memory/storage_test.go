package memory

import (
	"testing"

	. "gopkg.in/check.v1"
	"github.com/goabstract/go-git/storage/test"
)

func Test(t *testing.T) { TestingT(t) }

type StorageSuite struct {
	test.BaseStorageSuite
}

var _ = Suite(&StorageSuite{})

func (s *StorageSuite) SetUpTest(c *C) {
	s.BaseStorageSuite = test.NewBaseStorageSuite(NewStorage())
	s.BaseStorageSuite.SetUpTest(c)
}
