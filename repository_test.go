package git

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"srcd.works/go-git.v4/config"
	"github.com/src-d/go-git-fixtures"
	"srcd.works/go-git.v4/plumbing"
	"srcd.works/go-git.v4/plumbing/object"
	"srcd.works/go-git.v4/storage/filesystem"
	"srcd.works/go-git.v4/storage/memory"

	. "gopkg.in/check.v1"
	"srcd.works/go-billy.v1/memfs"
)

type RepositorySuite struct {
	BaseSuite
}

var _ = Suite(&RepositorySuite{})

func (s *RepositorySuite) TestInit(c *C) {
	r, err := Init(memory.NewStorage(), memfs.New())
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)

	cfg, err := r.Config()
	c.Assert(err, IsNil)
	c.Assert(cfg.Core.IsBare, Equals, false)
}

func (s *RepositorySuite) TestInitBare(c *C) {
	r, err := Init(memory.NewStorage(), nil)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)

	cfg, err := r.Config()
	c.Assert(err, IsNil)
	c.Assert(cfg.Core.IsBare, Equals, true)

}

func (s *RepositorySuite) TestInitAlreadyExists(c *C) {
	st := memory.NewStorage()

	r, err := Init(st, nil)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)

	r, err = Init(st, nil)
	c.Assert(err, Equals, ErrRepositoryAlreadyExists)
	c.Assert(r, IsNil)
}

func (s *RepositorySuite) TestOpen(c *C) {
	st := memory.NewStorage()

	r, err := Init(st, memfs.New())
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)

	r, err = Open(st, memfs.New())
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)
}

func (s *RepositorySuite) TestOpenBare(c *C) {
	st := memory.NewStorage()

	r, err := Init(st, nil)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)

	r, err = Open(st, nil)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)
}

func (s *RepositorySuite) TestOpenMissingWorktree(c *C) {
	st := memory.NewStorage()

	r, err := Init(st, memfs.New())
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)

	r, err = Open(st, nil)
	c.Assert(err, Equals, ErrWorktreeNotProvided)
	c.Assert(r, IsNil)
}

func (s *RepositorySuite) TestOpenNotExists(c *C) {
	r, err := Open(memory.NewStorage(), nil)
	c.Assert(err, Equals, ErrRepositoryNotExists)
	c.Assert(r, IsNil)
}

func (s *RepositorySuite) TestClone(c *C) {
	r, err := Clone(memory.NewStorage(), nil, &CloneOptions{
		URL: s.GetBasicLocalRepositoryURL(),
	})

	c.Assert(err, IsNil)

	remotes, err := r.Remotes()
	c.Assert(err, IsNil)
	c.Assert(remotes, HasLen, 1)
}

func (s *RepositorySuite) TestCreateRemoteAndRemote(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	remote, err := r.CreateRemote(&config.RemoteConfig{
		Name: "foo",
		URL:  "http://foo/foo.git",
	})

	c.Assert(err, IsNil)
	c.Assert(remote.Config().Name, Equals, "foo")

	alt, err := r.Remote("foo")
	c.Assert(err, IsNil)
	c.Assert(alt, Not(Equals), remote)
	c.Assert(alt.Config().Name, Equals, "foo")
}

func (s *RepositorySuite) TestCreateRemoteInvalid(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	remote, err := r.CreateRemote(&config.RemoteConfig{})

	c.Assert(err, Equals, config.ErrRemoteConfigEmptyName)
	c.Assert(remote, IsNil)
}

func (s *RepositorySuite) TestDeleteRemote(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	_, err := r.CreateRemote(&config.RemoteConfig{
		Name: "foo",
		URL:  "http://foo/foo.git",
	})

	c.Assert(err, IsNil)

	err = r.DeleteRemote("foo")
	c.Assert(err, IsNil)

	alt, err := r.Remote("foo")
	c.Assert(err, Equals, ErrRemoteNotFound)
	c.Assert(alt, IsNil)
}

func (s *RepositorySuite) TestPlainInit(c *C) {
	dir, err := ioutil.TempDir("", "plain-init")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := PlainInit(dir, true)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)

	cfg, err := r.Config()
	c.Assert(err, IsNil)
	c.Assert(cfg.Core.IsBare, Equals, true)
}

func (s *RepositorySuite) TestPlainInitAlreadyExists(c *C) {
	dir, err := ioutil.TempDir("", "plain-init")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := PlainInit(dir, true)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)

	r, err = PlainInit(dir, true)
	c.Assert(err, Equals, ErrRepositoryAlreadyExists)
	c.Assert(r, IsNil)
}

func (s *RepositorySuite) TestPlainOpen(c *C) {
	dir, err := ioutil.TempDir("", "plain-open")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := PlainInit(dir, false)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)

	r, err = PlainOpen(dir)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)
}

func (s *RepositorySuite) TestPlainOpenBare(c *C) {
	dir, err := ioutil.TempDir("", "plain-open")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := PlainInit(dir, true)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)

	r, err = PlainOpen(dir)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)
}

func (s *RepositorySuite) TestPlainOpenNotBare(c *C) {
	dir, err := ioutil.TempDir("", "plain-open")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := PlainInit(dir, false)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)

	r, err = PlainOpen(filepath.Join(dir, ".git"))
	c.Assert(err, Equals, ErrWorktreeNotProvided)
	c.Assert(r, IsNil)
}

func (s *RepositorySuite) TestPlainOpenNotExists(c *C) {
	r, err := PlainOpen("/not-exists/")
	c.Assert(err, Equals, ErrRepositoryNotExists)
	c.Assert(r, IsNil)
}

func (s *RepositorySuite) TestPlainClone(c *C) {
	dir, err := ioutil.TempDir("", "plain-clone")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := PlainClone(dir, false, &CloneOptions{
		URL: s.GetBasicLocalRepositoryURL(),
	})

	c.Assert(err, IsNil)

	remotes, err := r.Remotes()
	c.Assert(err, IsNil)
	c.Assert(remotes, HasLen, 1)
}

func (s *RepositorySuite) TestFetch(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	_, err := r.CreateRemote(&config.RemoteConfig{
		Name: DefaultRemoteName,
		URL:  s.GetBasicLocalRepositoryURL(),
	})
	c.Assert(err, IsNil)
	c.Assert(r.Fetch(&FetchOptions{}), IsNil)

	remotes, err := r.Remotes()
	c.Assert(err, IsNil)
	c.Assert(remotes, HasLen, 1)

	_, err = r.Head()
	c.Assert(err, Equals, plumbing.ErrReferenceNotFound)

	branch, err := r.Reference("refs/remotes/origin/master", false)
	c.Assert(err, IsNil)
	c.Assert(branch, NotNil)
	c.Assert(branch.Type(), Equals, plumbing.HashReference)
	c.Assert(branch.Hash().String(), Equals, "6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
}

func (s *RepositorySuite) TestCloneDeep(c *C) {
	fs := memfs.New()
	r, _ := Init(memory.NewStorage(), fs)

	head, err := r.Head()
	c.Assert(err, Equals, plumbing.ErrReferenceNotFound)
	c.Assert(head, IsNil)

	err = r.clone(&CloneOptions{
		URL: s.GetBasicLocalRepositoryURL(),
	})

	c.Assert(err, IsNil)

	remotes, err := r.Remotes()
	c.Assert(err, IsNil)
	c.Assert(remotes, HasLen, 1)

	head, err = r.Reference(plumbing.HEAD, false)
	c.Assert(err, IsNil)
	c.Assert(head, NotNil)
	c.Assert(head.Type(), Equals, plumbing.SymbolicReference)
	c.Assert(head.Target().String(), Equals, "refs/heads/master")

	branch, err := r.Reference(head.Target(), false)
	c.Assert(err, IsNil)
	c.Assert(branch, NotNil)
	c.Assert(branch.Hash().String(), Equals, "6ecf0ef2c2dffb796033e5a02219af86ec6584e5")

	branch, err = r.Reference("refs/remotes/origin/master", false)
	c.Assert(err, IsNil)
	c.Assert(branch, NotNil)
	c.Assert(branch.Type(), Equals, plumbing.HashReference)
	c.Assert(branch.Hash().String(), Equals, "6ecf0ef2c2dffb796033e5a02219af86ec6584e5")

	fi, err := fs.ReadDir("")
	c.Assert(err, IsNil)
	c.Assert(fi, HasLen, 8)
}

func (s *RepositorySuite) TestCloneConfig(c *C) {
	r, _ := Init(memory.NewStorage(), nil)

	head, err := r.Head()
	c.Assert(err, Equals, plumbing.ErrReferenceNotFound)
	c.Assert(head, IsNil)

	err = r.clone(&CloneOptions{
		URL: s.GetBasicLocalRepositoryURL(),
	})

	c.Assert(err, IsNil)

	cfg, err := r.Config()
	c.Assert(err, IsNil)

	c.Assert(cfg.Core.IsBare, Equals, true)
	c.Assert(cfg.Remotes, HasLen, 1)
	c.Assert(cfg.Remotes["origin"].Name, Equals, "origin")
	c.Assert(cfg.Remotes["origin"].URL, Not(Equals), "")
}

func (s *RepositorySuite) TestCloneSingleBranchAndNonHEAD(c *C) {
	r, _ := Init(memory.NewStorage(), nil)

	head, err := r.Head()
	c.Assert(err, Equals, plumbing.ErrReferenceNotFound)
	c.Assert(head, IsNil)

	err = r.clone(&CloneOptions{
		URL:           s.GetBasicLocalRepositoryURL(),
		ReferenceName: plumbing.ReferenceName("refs/heads/branch"),
		SingleBranch:  true,
	})

	c.Assert(err, IsNil)

	remotes, err := r.Remotes()
	c.Assert(err, IsNil)
	c.Assert(remotes, HasLen, 1)

	head, err = r.Reference(plumbing.HEAD, false)
	c.Assert(err, IsNil)
	c.Assert(head, NotNil)
	c.Assert(head.Type(), Equals, plumbing.SymbolicReference)
	c.Assert(head.Target().String(), Equals, "refs/heads/branch")

	branch, err := r.Reference(head.Target(), false)
	c.Assert(err, IsNil)
	c.Assert(branch, NotNil)
	c.Assert(branch.Hash().String(), Equals, "e8d3ffab552895c19b9fcf7aa264d277cde33881")

	branch, err = r.Reference("refs/remotes/origin/branch", false)
	c.Assert(err, IsNil)
	c.Assert(branch, NotNil)
	c.Assert(branch.Type(), Equals, plumbing.HashReference)
	c.Assert(branch.Hash().String(), Equals, "e8d3ffab552895c19b9fcf7aa264d277cde33881")
}

func (s *RepositorySuite) TestCloneSingleBranch(c *C) {
	r, _ := Init(memory.NewStorage(), nil)

	head, err := r.Head()
	c.Assert(err, Equals, plumbing.ErrReferenceNotFound)
	c.Assert(head, IsNil)

	err = r.clone(&CloneOptions{
		URL:          s.GetBasicLocalRepositoryURL(),
		SingleBranch: true,
	})

	c.Assert(err, IsNil)

	remotes, err := r.Remotes()
	c.Assert(err, IsNil)
	c.Assert(remotes, HasLen, 1)

	head, err = r.Reference(plumbing.HEAD, false)
	c.Assert(err, IsNil)
	c.Assert(head, NotNil)
	c.Assert(head.Type(), Equals, plumbing.SymbolicReference)
	c.Assert(head.Target().String(), Equals, "refs/heads/master")

	branch, err := r.Reference(head.Target(), false)
	c.Assert(err, IsNil)
	c.Assert(branch, NotNil)
	c.Assert(branch.Hash().String(), Equals, "6ecf0ef2c2dffb796033e5a02219af86ec6584e5")

	branch, err = r.Reference("refs/remotes/origin/master", false)
	c.Assert(err, IsNil)
	c.Assert(branch, NotNil)
	c.Assert(branch.Type(), Equals, plumbing.HashReference)
	c.Assert(branch.Hash().String(), Equals, "6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
}

func (s *RepositorySuite) TestCloneDetachedHEAD(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{
		URL:           s.GetBasicLocalRepositoryURL(),
		ReferenceName: plumbing.ReferenceName("refs/tags/v1.0.0"),
	})

	head, err := r.Reference(plumbing.HEAD, false)
	c.Assert(err, IsNil)
	c.Assert(head, NotNil)
	c.Assert(head.Type(), Equals, plumbing.HashReference)
	c.Assert(head.Hash().String(), Equals, "6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
}

func (s *RepositorySuite) TestPullCheckout(c *C) {
	fs := memfs.New()
	r, _ := Init(memory.NewStorage(), fs)
	r.CreateRemote(&config.RemoteConfig{
		Name: DefaultRemoteName,
		URL:  s.GetBasicLocalRepositoryURL(),
	})

	err := r.Pull(&PullOptions{})
	c.Assert(err, IsNil)

	fi, err := fs.ReadDir("")
	c.Assert(err, IsNil)
	c.Assert(fi, HasLen, 8)
}

func (s *RepositorySuite) TestCloneWithProgress(c *C) {
	fs := memfs.New()

	buf := bytes.NewBuffer(nil)
	_, err := Clone(memory.NewStorage(), fs, &CloneOptions{
		URL:      s.GetBasicLocalRepositoryURL(),
		Progress: buf,
	})

	c.Assert(err, IsNil)
	c.Assert(buf.Len(), Not(Equals), 0)
}

func (s *RepositorySuite) TestPullUpdateReferencesIfNeeded(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	r.CreateRemote(&config.RemoteConfig{
		Name: DefaultRemoteName,
		URL:  s.GetBasicLocalRepositoryURL(),
	})

	err := r.Fetch(&FetchOptions{})
	c.Assert(err, IsNil)

	_, err = r.Reference("refs/heads/master", false)
	c.Assert(err, NotNil)

	err = r.Pull(&PullOptions{})
	c.Assert(err, IsNil)

	head, err := r.Reference(plumbing.HEAD, true)
	c.Assert(err, IsNil)
	c.Assert(head.Hash().String(), Equals, "6ecf0ef2c2dffb796033e5a02219af86ec6584e5")

	branch, err := r.Reference("refs/heads/master", false)
	c.Assert(err, IsNil)
	c.Assert(branch.Hash().String(), Equals, "6ecf0ef2c2dffb796033e5a02219af86ec6584e5")

	err = r.Pull(&PullOptions{})
	c.Assert(err, Equals, NoErrAlreadyUpToDate)
}

func (s *RepositorySuite) TestPullSingleBranch(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{
		URL:          s.GetBasicLocalRepositoryURL(),
		SingleBranch: true,
	})

	c.Assert(err, IsNil)

	err = r.Pull(&PullOptions{})
	c.Assert(err, Equals, NoErrAlreadyUpToDate)

	branch, err := r.Reference("refs/heads/master", false)
	c.Assert(err, IsNil)
	c.Assert(branch.Hash().String(), Equals, "6ecf0ef2c2dffb796033e5a02219af86ec6584e5")

	branch, err = r.Reference("refs/remotes/foo/branch", false)
	c.Assert(err, NotNil)

	storage := r.s.(*memory.Storage)
	c.Assert(storage.Objects, HasLen, 28)
}

func (s *RepositorySuite) TestPullProgress(c *C) {
	r, _ := Init(memory.NewStorage(), nil)

	r.CreateRemote(&config.RemoteConfig{
		Name: DefaultRemoteName,
		URL:  s.GetBasicLocalRepositoryURL(),
	})

	buf := bytes.NewBuffer(nil)
	err := r.Pull(&PullOptions{
		Progress: buf,
	})

	c.Assert(err, IsNil)
	c.Assert(buf.Len(), Not(Equals), 0)
}

func (s *RepositorySuite) TestPullAdd(c *C) {
	path := fixtures.Basic().One().Worktree().Base()

	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{
		URL: fmt.Sprintf("file://%s", filepath.Join(path, ".git")),
	})

	c.Assert(err, IsNil)

	storage := r.s.(*memory.Storage)
	c.Assert(storage.Objects, HasLen, 31)

	branch, err := r.Reference("refs/heads/master", false)
	c.Assert(err, IsNil)
	c.Assert(branch.Hash().String(), Equals, "6ecf0ef2c2dffb796033e5a02219af86ec6584e5")

	branch, err = r.Reference("refs/remotes/origin/branch", false)
	c.Assert(err, IsNil)
	c.Assert(branch.Hash().String(), Equals, "e8d3ffab552895c19b9fcf7aa264d277cde33881")

	ExecuteOnPath(c, path,
		"touch foo",
		"git add foo",
		"git commit -m foo foo",
	)

	err = r.Pull(&PullOptions{RemoteName: "origin"})
	c.Assert(err, IsNil)

	// the commit command has introduced a new commit, tree and blob
	c.Assert(storage.Objects, HasLen, 34)

	branch, err = r.Reference("refs/heads/master", false)
	c.Assert(err, IsNil)
	c.Assert(branch.Hash().String(), Not(Equals), "6ecf0ef2c2dffb796033e5a02219af86ec6584e5")

	// the commit command, was in the local branch, so the remote should be read ok
	branch, err = r.Reference("refs/remotes/origin/branch", false)
	c.Assert(err, IsNil)
	c.Assert(branch.Hash().String(), Equals, "e8d3ffab552895c19b9fcf7aa264d277cde33881")
}

func (s *RepositorySuite) TestPushToEmptyRepository(c *C) {
	srcFs := fixtures.Basic().One().DotGit()
	sto, err := filesystem.NewStorage(srcFs)
	c.Assert(err, IsNil)

	dstFs := fixtures.ByTag("empty").One().DotGit()
	url := fmt.Sprintf("file://%s", dstFs.Base())

	r, err := Open(sto, srcFs)
	c.Assert(err, IsNil)

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "myremote",
		URL:  url,
	})
	c.Assert(err, IsNil)

	err = r.Push(&PushOptions{RemoteName: "myremote"})
	c.Assert(err, IsNil)

	sto, err = filesystem.NewStorage(dstFs)
	c.Assert(err, IsNil)
	dstRepo, err := Open(sto, nil)
	c.Assert(err, IsNil)

	iter, err := sto.IterReferences()
	c.Assert(err, IsNil)
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if !ref.IsBranch() {
			return nil
		}

		dstRef, err := dstRepo.Reference(ref.Name(), true)
		c.Assert(err, IsNil)
		c.Assert(dstRef, DeepEquals, ref)

		return nil
	})
	c.Assert(err, IsNil)
}

func (s *RepositorySuite) TestPushNonExistentRemote(c *C) {
	srcFs := fixtures.Basic().One().DotGit()
	sto, err := filesystem.NewStorage(srcFs)
	c.Assert(err, IsNil)

	r, err := Open(sto, srcFs)
	c.Assert(err, IsNil)

	err = r.Push(&PushOptions{RemoteName: "myremote"})
	c.Assert(err, ErrorMatches, ".*remote not found.*")
}

func (s *RepositorySuite) TestCommit(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{
		URL: s.GetBasicLocalRepositoryURL(),
	})

	c.Assert(err, IsNil)

	hash := plumbing.NewHash("b8e471f58bcbca63b07bda20e428190409c2db47")
	commit, err := r.Commit(hash)
	c.Assert(err, IsNil)

	c.Assert(commit.Hash.IsZero(), Equals, false)
	c.Assert(commit.Hash, Equals, commit.ID())
	c.Assert(commit.Hash, Equals, hash)
	c.Assert(commit.Type(), Equals, plumbing.CommitObject)

	tree, err := commit.Tree()
	c.Assert(err, IsNil)
	c.Assert(tree.Hash.IsZero(), Equals, false)

	c.Assert(commit.Author.Email, Equals, "daniel@lordran.local")
}

func (s *RepositorySuite) TestCommits(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{URL: s.GetBasicLocalRepositoryURL()})
	c.Assert(err, IsNil)

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
		c.Assert(commit.Type(), Equals, plumbing.CommitObject)
	}

	c.Assert(count, Equals, 9)
}

func (s *RepositorySuite) TestTag(c *C) {
	url := s.GetLocalRepositoryURL(
		fixtures.ByURL("https://github.com/git-fixtures/tags.git").One(),
	)

	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{URL: url})
	c.Assert(err, IsNil)

	hash := plumbing.NewHash("ad7897c0fb8e7d9a9ba41fa66072cf06095a6cfc")
	tag, err := r.Tag(hash)
	c.Assert(err, IsNil)

	c.Assert(tag.Hash.IsZero(), Equals, false)
	c.Assert(tag.Hash, Equals, hash)
	c.Assert(tag.Type(), Equals, plumbing.TagObject)
}

func (s *RepositorySuite) TestTags(c *C) {
	url := s.GetLocalRepositoryURL(
		fixtures.ByURL("https://github.com/git-fixtures/tags.git").One(),
	)

	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{URL: url})
	c.Assert(err, IsNil)

	count := 0
	tags, err := r.Tags()
	c.Assert(err, IsNil)

	tags.ForEach(func(tag *object.Tag) error {
		count++

		c.Assert(tag.Hash.IsZero(), Equals, false)
		c.Assert(tag.Type(), Equals, plumbing.TagObject)
		return nil
	})

	refs, _ := r.References()
	refs.ForEach(func(ref *plumbing.Reference) error {
		return nil
	})

	c.Assert(count, Equals, 4)
}

func (s *RepositorySuite) TestCommitIterClosePanic(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{URL: s.GetBasicLocalRepositoryURL()})
	c.Assert(err, IsNil)

	commits, err := r.Commits()
	c.Assert(err, IsNil)
	commits.Close()
}

func (s *RepositorySuite) TestRef(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{URL: s.GetBasicLocalRepositoryURL()})
	c.Assert(err, IsNil)

	ref, err := r.Reference(plumbing.HEAD, false)
	c.Assert(err, IsNil)
	c.Assert(ref.Name(), Equals, plumbing.HEAD)

	ref, err = r.Reference(plumbing.HEAD, true)
	c.Assert(err, IsNil)
	c.Assert(ref.Name(), Equals, plumbing.ReferenceName("refs/heads/master"))
}

func (s *RepositorySuite) TestRefs(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{URL: s.GetBasicLocalRepositoryURL()})
	c.Assert(err, IsNil)

	c.Assert(err, IsNil)

	iter, err := r.References()
	c.Assert(err, IsNil)
	c.Assert(iter, NotNil)
}

func (s *RepositorySuite) TestObject(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{URL: s.GetBasicLocalRepositoryURL()})
	c.Assert(err, IsNil)

	hash := plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
	o, err := r.Object(plumbing.CommitObject, hash)
	c.Assert(err, IsNil)

	c.Assert(o.ID().IsZero(), Equals, false)
	c.Assert(o.Type(), Equals, plumbing.CommitObject)
}

func (s *RepositorySuite) TestObjectNotFound(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	err := r.clone(&CloneOptions{URL: s.GetBasicLocalRepositoryURL()})
	c.Assert(err, IsNil)

	hash := plumbing.NewHash("0a3fb06ff80156fb153bcdcc58b5e16c2d27625c")
	tag, err := r.Object(plumbing.TagObject, hash)
	c.Assert(err, DeepEquals, plumbing.ErrObjectNotFound)
	c.Assert(tag, IsNil)
}

func (s *RepositorySuite) TestWorktree(c *C) {
	def := memfs.New()
	r, _ := Init(memory.NewStorage(), def)
	w, err := r.Worktree()
	c.Assert(err, IsNil)
	c.Assert(w.fs, Equals, def)
}

func (s *RepositorySuite) TestWorktreeBare(c *C) {
	r, _ := Init(memory.NewStorage(), nil)
	w, err := r.Worktree()
	c.Assert(err, Equals, ErrIsBareRepository)
	c.Assert(w, IsNil)
}

func (s *RepositorySuite) TestResolveRevision(c *C) {
	url := s.GetLocalRepositoryURL(
		fixtures.ByURL("https://github.com/git-fixtures/basic.git").One(),
	)

	r := NewMemoryRepository()
	err := r.Clone(&CloneOptions{URL: url})
	c.Assert(err, IsNil)

	datas := map[string]string{
		"HEAD": "6ecf0ef2c2dffb796033e5a02219af86ec6584e5",
		"refs/heads/master~2^^~": "b029517f6300c2da0f4b651b8642506cd6aaf45d",
		"HEAD~2^^~":              "b029517f6300c2da0f4b651b8642506cd6aaf45d",
		"HEAD~3^2":               "a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69",
		"HEAD~3^2^0":             "a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69",
		"HEAD~2^{/binary file}":  "35e85108805c84807bc66a02d91535e1e24b38b9",
	}

	for rev, hash := range datas {
		h, err := r.ResolveRevision(plumbing.Revision(rev))

		c.Assert(err, IsNil)
		c.Assert(h.String(), Equals, hash)
	}
}

func (s *RepositorySuite) TestResolveRevisionWithErrors(c *C) {
	url := s.GetLocalRepositoryURL(
		fixtures.ByURL("https://github.com/git-fixtures/basic.git").One(),
	)

	r := NewMemoryRepository()
	err := r.Clone(&CloneOptions{URL: url})
	c.Assert(err, IsNil)

	datas := map[string]string{
		"efs/heads/master~": "reference not found",
		"HEAD^3":            `Revision invalid : "3" found must be 0, 1 or 2 after "^"`,
		"HEAD^{/whatever}":  `No commit message match regexp : "whatever"`,
	}

	for rev, rerr := range datas {
		_, err := r.ResolveRevision(plumbing.Revision(rev))

		c.Assert(err.Error(), Equals, rerr)
	}
}

func ExecuteOnPath(c *C, path string, cmds ...string) error {
	for _, cmd := range cmds {
		err := executeOnPath(path, cmd)
		c.Assert(err, IsNil)
	}

	return nil
}

func executeOnPath(path, cmd string) error {
	args := strings.Split(cmd, " ")
	c := exec.Command(args[0], args[1:]...)
	c.Dir = path
	c.Env = os.Environ()

	buf := bytes.NewBuffer(nil)
	c.Stderr = buf
	c.Stdout = buf

	//defer func() { fmt.Println(buf.String()) }()

	return c.Run()
}
