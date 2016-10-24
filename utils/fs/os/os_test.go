package os_test

import (
	"io/ioutil"
	"os"
	"testing"

	. "gopkg.in/check.v1"
	osfs "gopkg.in/src-d/go-git.v4/utils/fs/os"
	"gopkg.in/src-d/go-git.v4/utils/fs/test"
)

func Test(t *testing.T) { TestingT(t) }

type OSSuite struct {
	test.FilesystemSuite
	path string
}

var _ = Suite(&OSSuite{})

func (s *OSSuite) SetUpTest(c *C) {
	s.path, _ = ioutil.TempDir(os.TempDir(), "go-git-os-fs-test")
	s.FilesystemSuite.Fs = osfs.NewOS(s.path)
}
func (s *OSSuite) TearDownTest(c *C) {
	err := os.RemoveAll(s.path)
	c.Assert(err, IsNil)
}
