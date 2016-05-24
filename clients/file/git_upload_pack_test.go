package file

import (
	"io/ioutil"
	"os"
	"testing"

	"gopkg.in/src-d/go-git.v3/clients/common"
	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/utils/tgz"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteFile struct {
	dirRemotePath string
}

var _ = Suite(&SuiteFile{})

const repositoryFixture = "../../formats/gitdir/fixtures/spinnaker-gc.tgz"

func (s *SuiteFile) SetUpSuite(c *C) {
	file, err := os.Open(repositoryFixture)
	c.Assert(err, IsNil)

	defer func() {
		err := file.Close()
		c.Assert(err, IsNil)
	}()

	s.dirRemotePath, err = tgz.Extract(file)
	c.Assert(err, IsNil)
}

func (s *SuiteFile) TearDownSuite(c *C) {
	err := os.RemoveAll(s.dirRemotePath)
	c.Assert(err, IsNil)
}

func (s *SuiteFile) TestConnect(c *C) {
	r := NewGitUploadPackService()
	err := r.Connect(repositoryFixture)
	c.Assert(err, IsNil)
}

func (s *SuiteFile) TestConnectWithAuth(c *C) {
	r := NewGitUploadPackService()
	err := r.ConnectWithAuth(repositoryFixture, nil)
	c.Assert(err, IsNil)

	r = NewGitUploadPackService()
	auth := dummyAuth{}
	err = r.ConnectWithAuth(repositoryFixture, auth)
	c.Assert(err, Equals, common.ErrAuthNotSupported)
}

type dummyAuth struct{}

func (d dummyAuth) Name() string   { return "" }
func (d dummyAuth) String() string { return "" }

func (s *SuiteFile) TestDefaultBranch(c *C) {
	r := NewGitUploadPackService()
	err := r.Connect(repositoryFixture)
	c.Assert(err, IsNil)

	info, err := r.Info()
	c.Assert(err, IsNil)
	c.Assert(info.Capabilities.SymbolicReference("HEAD"), Equals, "refs/heads/master")
}

func (s *SuiteFile) TestFetch(c *C) {
	r := NewGitUploadPackService()
	c.Assert(r.Connect(repositoryFixture), IsNil)

	req := &common.GitUploadPackRequest{}
	req.Want(core.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))

	reader, err := r.Fetch(req)
	c.Assert(err, IsNil)

	b, err := ioutil.ReadAll(reader)
	c.Assert(err, IsNil)
	c.Assert(b, HasLen, 85374)
}
