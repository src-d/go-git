package git

import (
	"errors"
	"io"
	"os"
	"testing"

	"gopkg.in/src-d/go-git.v4/clients"
	"gopkg.in/src-d/go-git.v4/clients/common"
	"gopkg.in/src-d/go-git.v4/core"
	"gopkg.in/src-d/go-git.v4/fixtures"
	"gopkg.in/src-d/go-git.v4/formats/packfile"
	"gopkg.in/src-d/go-git.v4/formats/packp"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type BaseSuite struct {
	fixtures.Suite

	Repository   *Repository
	Repositories map[string]*Repository
}

func (s *BaseSuite) SetUpSuite(c *C) {
	s.Suite.SetUpSuite(c)
	s.installMockProtocol(c)
	s.buildRepository(c)

	s.Repositories = make(map[string]*Repository, 0)
	s.buildRepositories(c, fixtures.Basic().ByTag("packfile"))
}

func (s *BaseSuite) installMockProtocol(c *C) {
	clients.InstallProtocol("https", func(end common.Endpoint) common.GitUploadPackService {
		return &MockGitUploadPackService{endpoint: end}
	})
}

func (s *BaseSuite) buildRepository(c *C) {
	f := fixtures.Basic().One()

	var err error
	s.Repository, err = NewFilesystemRepository(f.DotGit().Base())
	c.Assert(err, IsNil)
}

func (s *BaseSuite) buildRepositories(c *C, f fixtures.Fixtures) {
	for _, fixture := range f {
		r := NewMemoryRepository()

		f := fixture.Packfile()
		defer f.Close()

		n := packfile.NewScanner(f)
		d, err := packfile.NewDecoder(n, r.s.ObjectStorage())
		c.Assert(err, IsNil)
		_, err = d.Decode()
		c.Assert(err, IsNil)

		s.Repositories[fixture.URL] = r
	}
}

const RepositoryFixture = "https://github.com/git-fixtures/basic.git"

type MockGitUploadPackService struct {
	connected bool
	endpoint  common.Endpoint
	auth      common.AuthMethod
}

func (p *MockGitUploadPackService) Connect() error {
	p.connected = true
	return nil
}

func (p *MockGitUploadPackService) SetAuth(auth common.AuthMethod) error {
	p.auth = auth
	return nil
}

func (p *MockGitUploadPackService) Info() (*common.GitUploadPackInfo, error) {
	if !p.connected {
		return nil, errors.New("not connected")
	}

	h := fixtures.ByURL(p.endpoint.String()).One().Head

	c := packp.NewCapabilities()
	c.Decode("6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEADmulti_ack thin-pack side-band side-band-64k ofs-delta shallow no-progress include-tag multi_ack_detailed no-done symref=HEAD:refs/heads/master agent=git/2:2.4.8~dbussink-fix-enterprise-tokens-compilation-1167-gc7006cf")

	ref := core.ReferenceName("refs/heads/master")
	branch := core.ReferenceName("refs/heads/branch")
	tag := core.ReferenceName("refs/tags/v1.0.0")
	return &common.GitUploadPackInfo{
		Capabilities: c,
		Refs: map[core.ReferenceName]*core.Reference{
			core.HEAD: core.NewSymbolicReference(core.HEAD, ref),
			ref:       core.NewHashReference(ref, h),
			tag:       core.NewHashReference(tag, h),
			branch:    core.NewHashReference(branch, core.NewHash("e8d3ffab552895c19b9fcf7aa264d277cde33881")),
		},
	}, nil
}

func (p *MockGitUploadPackService) Fetch(r *common.GitUploadPackRequest) (io.ReadCloser, error) {
	if !p.connected {
		return nil, errors.New("not connected")
	}

	f := fixtures.ByURL(p.endpoint.String())

	if len(r.Wants) == 1 {
		return f.Exclude("single-branch").One().Packfile(), nil
	}

	return f.One().Packfile(), nil
}

func (p *MockGitUploadPackService) Disconnect() error {
	p.connected = false
	return nil
}

type packedFixture struct {
	url      string
	packfile string
}

var fixtureRepos = []packedFixture{
	{"https://github.com/tyba/git-fixture.git", "formats/packfile/fixtures/git-fixture.ofs-delta"},
	{"https://github.com/jamesob/desk.git", "formats/packfile/fixtures/jamesob-desk.pack"},
	{"https://github.com/spinnaker/spinnaker.git", "formats/packfile/fixtures/spinnaker-spinnaker.pack"},
}

func unpackFixtures(c *C, fixtures ...[]packedFixture) map[string]*Repository {
	repos := make(map[string]*Repository, 0)
	for _, group := range fixtures {
		for _, fixture := range group {
			if _, existing := repos[fixture.url]; existing {
				continue
			}

			comment := Commentf("fixture packfile: %q", fixture.packfile)

			repos[fixture.url] = NewMemoryRepository()

			f, err := os.Open(fixture.packfile)
			c.Assert(err, IsNil, comment)

			r := packfile.NewScanner(f)
			d, err := packfile.NewDecoder(r, repos[fixture.url].s.ObjectStorage())
			c.Assert(err, IsNil, comment)
			_, err = d.Decode()
			c.Assert(err, IsNil, comment)
			c.Assert(f.Close(), IsNil, comment)
		}
	}

	return repos
}

type SuiteCommon struct{}

var _ = Suite(&SuiteCommon{})

var countLinesTests = [...]struct {
	i string // the string we want to count lines from
	e int    // the expected number of lines in i
}{
	{"", 0},
	{"a", 1},
	{"a\n", 1},
	{"a\nb", 2},
	{"a\nb\n", 2},
	{"a\nb\nc", 3},
	{"a\nb\nc\n", 3},
	{"a\n\n\nb\n", 4},
	{"first line\n\tsecond line\nthird line\n", 3},
}

func (s *SuiteCommon) TestCountLines(c *C) {
	for i, t := range countLinesTests {
		o := countLines(t.i)
		c.Assert(o, Equals, t.e, Commentf("subtest %d, input=%q", i, t.i))
	}
}

func (s *BaseSuite) Clone(url string) *Repository {
	r := NewMemoryRepository()
	if err := r.Clone(&CloneOptions{URL: url}); err != nil {
		panic(err)
	}

	return r
}

func (s *BaseSuite) NewRepository(f *fixtures.Fixture) *Repository {
	storage, err := filesystem.NewStorage(f.DotGit())
	if err != nil {
		panic(err)
	}

	r, err := NewRepository(storage)
	if err != nil {
		panic(err)
	}

	return r
}
