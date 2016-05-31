package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/src-d/go-git.v3/clients/common"
	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/utils/tgz"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteFileClient struct {
	fixtureURL  common.Endpoint
	fixturePath string
}

var _ = Suite(&SuiteFileClient{})

const fixtureTGZ = "../../formats/gitdir/fixtures/spinnaker-gc.tgz"

func (s *SuiteFileClient) SetUpSuite(c *C) {
	var err error

	s.fixturePath, err = tgz.Extract(fixtureTGZ)
	c.Assert(err, IsNil)

	s.fixtureURL = common.Endpoint("file://" +
		filepath.Join(s.fixturePath, ".git"))
}

func (s *SuiteFileClient) TearDownSuite(c *C) {
	err := os.RemoveAll(s.fixturePath)
	c.Assert(err, IsNil)
}

func (s *SuiteFileClient) TestConnect(c *C) {
	r := NewGitUploadPackService()
	err := r.Connect(s.fixtureURL)
	c.Assert(err, IsNil)
}

func (s *SuiteFileClient) TestConnectWithAuth(c *C) {
	r := NewGitUploadPackService()
	err := r.ConnectWithAuth(s.fixtureURL, nil)
	c.Assert(err, IsNil)

	r = NewGitUploadPackService()
	auth := dummyAuth{}
	err = r.ConnectWithAuth(s.fixtureURL, auth)
	c.Assert(err, Equals, common.ErrAuthNotSupported)
}

type dummyAuth struct{}

func (d dummyAuth) Name() string   { return "" }
func (d dummyAuth) String() string { return "" }

func (s *SuiteFileClient) TestDefaultBranch(c *C) {
	r := NewGitUploadPackService()
	err := r.Connect(s.fixtureURL)
	c.Assert(err, IsNil)

	info, err := r.Info()
	c.Assert(err, IsNil)
	c.Assert(info.Capabilities.SymbolicReference("HEAD"), Equals, "refs/heads/master")
}

func (s *SuiteFileClient) NoTestFetch(c *C) {
	r := NewGitUploadPackService()
	c.Assert(r.Connect(s.fixtureURL), IsNil)

	req := &common.GitUploadPackRequest{}
	req.Want(core.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))

	reader, err := r.Fetch(req)
	c.Assert(err, IsNil)

	b, err := ioutil.ReadAll(reader)
	c.Assert(err, IsNil)
	c.Assert(b, HasLen, 85374)
}
