package git

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/src-d/go-git.v3/clients/http"
	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/storage/seekable"
	"gopkg.in/src-d/go-git.v3/utils/tgz"

	. "gopkg.in/check.v1"
)

var dirFixtures = [...]struct {
	name string
	tgz  string
}{
	{
		name: "spinnaker",
		tgz:  "formats/gitdir/fixtures/spinnaker-gc.tgz",
	},
}

type SuiteRepository struct {
	repos           map[string]*Repository
	dirFixturePaths map[string]string
}

var _ = Suite(&SuiteRepository{})

func (s *SuiteRepository) SetUpSuite(c *C) {
	s.repos = unpackFixtures(c, tagFixtures, treeWalkerFixtures)

	s.dirFixturePaths = make(map[string]string, len(dirFixtures))
	for _, fixture := range dirFixtures {
		comment := Commentf("fixture name = %s\n", fixture.name)

		path, err := tgz.Extract(fixture.tgz)
		c.Assert(err, IsNil, comment)

		s.dirFixturePaths[fixture.name] = filepath.Join(path, ".git")
	}
}

func (s *SuiteRepository) TearDownSuite(c *C) {
	for name, path := range s.dirFixturePaths {
		err := os.RemoveAll(path)
		c.Assert(err, IsNil, Commentf("cannot delete tmp dir for fixture %s: %s\n",
			name, path))
	}
}

func (s *SuiteRepository) TestNewRepository(c *C) {
	r, err := NewRepository(RepositoryFixture, nil)
	c.Assert(err, IsNil)
	c.Assert(r.Remotes["origin"].Auth, IsNil)
	c.Assert(r.URL, Equals, RepositoryFixture)
}

func (s *SuiteRepository) TestNewRepositoryWithAuth(c *C) {
	auth := &http.BasicAuth{}
	r, err := NewRepository(RepositoryFixture, auth)
	c.Assert(err, IsNil)
	c.Assert(r.Remotes["origin"].Auth, Equals, auth)
}

func (s *SuiteRepository) TestNewSeekableRepository(c *C) {
	for name, path := range s.dirFixturePaths {
		comment := Commentf("dir fixture %q → %q\n", name, path)
		repo, err := NewRepository("local://"+path, nil)
		c.Assert(err, IsNil, comment)

		err = repo.PullDefault()
		c.Assert(err, IsNil, comment)

		c.Assert(repo.Storage, NotNil, comment)
		c.Assert(repo.Storage, FitsTypeOf, &seekable.ObjectStorage{}, comment)
	}
}

func (s *SuiteRepository) TestPull(c *C) {
	r, err := NewRepository(RepositoryFixture, nil)
	r.Remotes["origin"].upSrv = &MockGitUploadPackService{}

	c.Assert(err, IsNil)
	c.Assert(r.Pull("origin", "refs/heads/master"), IsNil)

	mock, ok := (r.Remotes["origin"].upSrv).(*MockGitUploadPackService)
	c.Assert(ok, Equals, true)
	err = mock.RC.Close()
	c.Assert(err, Not(IsNil), Commentf("pull leaks an open fd from the fetch"))
}

func (s *SuiteRepository) TestPullDefault(c *C) {
	r, err := NewRepository(RepositoryFixture, nil)
	r.Remotes[DefaultRemoteName].Connect()
	r.Remotes[DefaultRemoteName].upSrv = &MockGitUploadPackService{}

	c.Assert(err, IsNil)
	c.Assert(r.PullDefault(), IsNil)

	mock, ok := (r.Remotes[DefaultRemoteName].upSrv).(*MockGitUploadPackService)
	c.Assert(ok, Equals, true)
	err = mock.RC.Close()
	c.Assert(err, Not(IsNil), Commentf("pull leaks an open fd from the fetch"))
}

func (s *SuiteRepository) TestCommit(c *C) {
	r, err := NewRepository(RepositoryFixture, nil)
	r.Remotes["origin"].upSrv = &MockGitUploadPackService{}

	c.Assert(err, IsNil)
	c.Assert(r.Pull("origin", "refs/heads/master"), IsNil)

	hash := core.NewHash("b8e471f58bcbca63b07bda20e428190409c2db47")
	commit, err := r.Commit(hash)
	c.Assert(err, IsNil)

	c.Assert(commit.Hash.IsZero(), Equals, false)
	c.Assert(commit.Hash, Equals, commit.ID())
	c.Assert(commit.Hash, Equals, hash)
	c.Assert(commit.Type(), Equals, core.CommitObject)
	c.Assert(commit.Tree().Hash.IsZero(), Equals, false)
	c.Assert(commit.Author.Email, Equals, "daniel@lordran.local")
}

func (s *SuiteRepository) TestCommits(c *C) {
	r, err := NewRepository(RepositoryFixture, nil)
	r.Remotes["origin"].upSrv = &MockGitUploadPackService{}

	c.Assert(err, IsNil)
	c.Assert(r.Pull("origin", "refs/heads/master"), IsNil)

	count := 0
	commits, err := r.Commits()
	c.Assert(err, IsNil)
	for {
		commit, err := commits.Next()
		if err != nil {
			break
		}

		count++
		c.Assert(commit.Hash.IsZero(), Equals, false)
		c.Assert(commit.Hash, Equals, commit.ID())
		c.Assert(commit.Type(), Equals, core.CommitObject)
		//c.Assert(commit.Tree.IsZero(), Equals, false)
	}

	c.Assert(count, Equals, 8)
}

func (s *SuiteRepository) TestTag(c *C) {
	for i, t := range tagTests {
		r, ok := s.repos[t.repo]
		c.Assert(ok, Equals, true)
		k := 0
		for hashString, expected := range t.tags {
			hash := core.NewHash(hashString)
			tag, err := r.Tag(hash)
			c.Assert(err, IsNil)
			testTagExpected(c, tag, hash, expected, fmt.Sprintf("subtest %d, tag %d: ", i, k))
			k++
		}
	}
}

func (s *SuiteRepository) TestTags(c *C) {
	for i, t := range tagTests {
		r, ok := s.repos[t.repo]
		c.Assert(ok, Equals, true)
		tagsIter, err := r.Tags()
		c.Assert(err, IsNil)
		testTagIter(c, tagsIter, t.tags, fmt.Sprintf("subtest %d, ", i))
	}
}

func (s *SuiteRepository) TestObject(c *C) {
	for i, t := range treeWalkerTests {
		r, ok := s.repos[t.repo]
		c.Assert(ok, Equals, true)
		for k := 0; k < len(t.objs); k++ {
			comment := fmt.Sprintf("subtest %d, tag %d", i, k)
			info := t.objs[k]
			hash := core.NewHash(info.Hash)
			obj, err := r.Object(hash)
			c.Assert(err, IsNil, Commentf(comment))
			c.Assert(obj.Type(), Equals, info.Kind, Commentf(comment))
			c.Assert(obj.ID(), Equals, hash, Commentf(comment))
		}
	}
}

func (s *SuiteRepository) TestCommitIterClosePanic(c *C) {
	r, err := NewRepository(RepositoryFixture, nil)
	r.Remotes["origin"].upSrv = &MockGitUploadPackService{}

	c.Assert(err, IsNil)
	c.Assert(r.Pull("origin", "refs/heads/master"), IsNil)

	commits, err := r.Commits()
	c.Assert(err, IsNil)
	commits.Close()
}
