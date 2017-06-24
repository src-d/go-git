package git

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/filemode"
	"gopkg.in/src-d/go-git.v4/plumbing/format/index"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"

	"github.com/src-d/go-git-fixtures"
	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-billy.v3/memfs"
	"gopkg.in/src-d/go-billy.v3/osfs"
	"gopkg.in/src-d/go-billy.v3/util"
)

type WorktreeSuite struct {
	BaseSuite
}

var _ = Suite(&WorktreeSuite{})

func (s *WorktreeSuite) SetUpTest(c *C) {
	s.buildBasicRepository(c)
	// the index is removed if not the Repository will be not clean
	c.Assert(s.Repository.Storer.SetIndex(&index.Index{Version: 2}), IsNil)
}

func (s *WorktreeSuite) TestCheckout(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	entries, err := fs.ReadDir("/")
	c.Assert(err, IsNil)

	c.Assert(entries, HasLen, 8)
	ch, err := fs.Open("CHANGELOG")
	c.Assert(err, IsNil)

	content, err := ioutil.ReadAll(ch)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "Initial changelog\n")

	idx, err := s.Repository.Storer.Index()
	c.Assert(err, IsNil)
	c.Assert(idx.Entries, HasLen, 9)
}

func (s *WorktreeSuite) TestCheckoutSymlink(c *C) {
	dir, err := ioutil.TempDir("", "checkout")
	defer os.RemoveAll(dir)

	r, err := PlainInit(dir, false)
	c.Assert(err, IsNil)

	w, err := r.Worktree()
	c.Assert(err, IsNil)

	w.fs.Symlink("not-exists", "bar")
	w.Add("bar")
	w.Commit("foo", &CommitOptions{Author: defaultSignature()})

	r.Storer.SetIndex(&index.Index{Version: 2})
	w.fs = osfs.New(filepath.Join(dir, "worktree-empty"))

	err = w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status.IsClean(), Equals, true)

	target, err := w.fs.Readlink("bar")
	c.Assert(target, Equals, "not-exists")
	c.Assert(err, IsNil)
}

func (s *WorktreeSuite) TestCheckoutSubmodule(c *C) {
	url := "https://github.com/git-fixtures/submodule.git"
	w := &Worktree{
		r:  s.NewRepository(fixtures.ByURL(url).One()),
		fs: memfs.New(),
	}

	// we delete the index, since the fixture comes with a real index
	err := w.r.Storer.SetIndex(&index.Index{Version: 2})
	c.Assert(err, IsNil)

	err = w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status.IsClean(), Equals, true)
}

func (s *WorktreeSuite) TestCheckoutSubmoduleInitialized(c *C) {
	url := "https://github.com/git-fixtures/submodule.git"
	w := &Worktree{
		r:  s.NewRepository(fixtures.ByURL(url).One()),
		fs: memfs.New(),
	}

	err := w.r.Storer.SetIndex(&index.Index{Version: 2})
	c.Assert(err, IsNil)

	err = w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)
	sub, err := w.Submodules()
	c.Assert(err, IsNil)

	err = sub.Update(&SubmoduleUpdateOptions{Init: true})
	c.Assert(err, IsNil)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status.IsClean(), Equals, true)
}

func (s *WorktreeSuite) TestCheckoutIndexMem(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	idx, err := s.Repository.Storer.Index()
	c.Assert(err, IsNil)
	c.Assert(idx.Entries, HasLen, 9)
	c.Assert(idx.Entries[0].Hash.String(), Equals, "32858aad3c383ed1ff0a0f9bdf231d54a00c9e88")
	c.Assert(idx.Entries[0].Name, Equals, ".gitignore")
	c.Assert(idx.Entries[0].Mode, Equals, filemode.Regular)
	c.Assert(idx.Entries[0].ModifiedAt.IsZero(), Equals, false)
	c.Assert(idx.Entries[0].Size, Equals, uint32(189))

	// ctime, dev, inode, uid and gid are not supported on memfs fs
	c.Assert(idx.Entries[0].CreatedAt.IsZero(), Equals, true)
	c.Assert(idx.Entries[0].Dev, Equals, uint32(0))
	c.Assert(idx.Entries[0].Inode, Equals, uint32(0))
	c.Assert(idx.Entries[0].UID, Equals, uint32(0))
	c.Assert(idx.Entries[0].GID, Equals, uint32(0))
}

func (s *WorktreeSuite) TestCheckoutIndexOS(c *C) {
	dir, err := ioutil.TempDir("", "checkout")
	defer os.RemoveAll(dir)

	fs := osfs.New(filepath.Join(dir, "worktree"))
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err = w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	idx, err := s.Repository.Storer.Index()
	c.Assert(err, IsNil)
	c.Assert(idx.Entries, HasLen, 9)
	c.Assert(idx.Entries[0].Hash.String(), Equals, "32858aad3c383ed1ff0a0f9bdf231d54a00c9e88")
	c.Assert(idx.Entries[0].Name, Equals, ".gitignore")
	c.Assert(idx.Entries[0].Mode, Equals, filemode.Regular)
	c.Assert(idx.Entries[0].ModifiedAt.IsZero(), Equals, false)
	c.Assert(idx.Entries[0].Size, Equals, uint32(189))

	c.Assert(idx.Entries[0].CreatedAt.IsZero(), Equals, false)
	c.Assert(idx.Entries[0].Dev, Not(Equals), uint32(0))
	c.Assert(idx.Entries[0].Inode, Not(Equals), uint32(0))
	c.Assert(idx.Entries[0].UID, Not(Equals), uint32(0))
	c.Assert(idx.Entries[0].GID, Not(Equals), uint32(0))
}

func (s *WorktreeSuite) TestCheckoutChange(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	head, err := w.r.Head()
	c.Assert(err, IsNil)
	c.Assert(head.Name().String(), Equals, "refs/heads/master")

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status.IsClean(), Equals, true)

	_, err = fs.Stat("README")
	c.Assert(err, Equals, os.ErrNotExist)
	_, err = fs.Stat("vendor")
	c.Assert(err, Equals, nil)

	err = w.Checkout(&CheckoutOptions{
		Branch: "refs/heads/branch",
	})
	c.Assert(err, IsNil)

	status, err = w.Status()
	c.Assert(err, IsNil)
	c.Assert(status.IsClean(), Equals, true)

	_, err = fs.Stat("README")
	c.Assert(err, Equals, nil)

	_, err = fs.Stat("vendor")
	c.Assert(err, Equals, os.ErrNotExist)

	head, err = w.r.Head()
	c.Assert(err, IsNil)
	c.Assert(head.Name().String(), Equals, "refs/heads/branch")
}

func (s *WorktreeSuite) TestCheckoutTag(c *C) {
	f := fixtures.ByTag("tags").One()

	fs := memfs.New()
	w := &Worktree{
		r:  s.NewRepository(f),
		fs: fs,
	}

	// we delete the index, since the fixture comes with a real index
	err := w.r.Storer.SetIndex(&index.Index{Version: 2})
	c.Assert(err, IsNil)

	err = w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)
	head, err := w.r.Head()
	c.Assert(err, IsNil)
	c.Assert(head.Name().String(), Equals, "refs/heads/master")

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status.IsClean(), Equals, true)

	err = w.Checkout(&CheckoutOptions{Branch: "refs/tags/lightweight-tag"})
	c.Assert(err, IsNil)
	head, err = w.r.Head()
	c.Assert(err, IsNil)
	c.Assert(head.Name().String(), Equals, "HEAD")
	c.Assert(head.Hash().String(), Equals, "f7b877701fbf855b44c0a9e86f3fdce2c298b07f")

	err = w.Checkout(&CheckoutOptions{Branch: "refs/tags/commit-tag"})
	c.Assert(err, IsNil)
	head, err = w.r.Head()
	c.Assert(err, IsNil)
	c.Assert(head.Name().String(), Equals, "HEAD")
	c.Assert(head.Hash().String(), Equals, "f7b877701fbf855b44c0a9e86f3fdce2c298b07f")

	err = w.Checkout(&CheckoutOptions{Branch: "refs/tags/tree-tag"})
	c.Assert(err, NotNil)
	head, err = w.r.Head()
	c.Assert(err, IsNil)
	c.Assert(head.Name().String(), Equals, "HEAD")
}

func (s *WorktreeSuite) TestCheckoutBisect(c *C) {
	s.testCheckoutBisect(c, "https://github.com/src-d/go-git.git")
}

func (s *WorktreeSuite) TestCheckoutBisectSubmodules(c *C) {
	s.testCheckoutBisect(c, "https://github.com/git-fixtures/submodule.git")
}

// TestCheckoutBisect simulates a git bisect going through the git history and
// checking every commit over the previous commit
func (s *WorktreeSuite) testCheckoutBisect(c *C, url string) {
	f := fixtures.ByURL(url).One()

	w := &Worktree{
		r:  s.NewRepository(f),
		fs: memfs.New(),
	}

	// we delete the index, since the fixture comes with a real index
	err := w.r.Storer.SetIndex(&index.Index{Version: 2})
	c.Assert(err, IsNil)

	iter, err := w.r.Log(&LogOptions{})
	c.Assert(err, IsNil)

	iter.ForEach(func(commit *object.Commit) error {
		err := w.Checkout(&CheckoutOptions{Hash: commit.Hash})
		c.Assert(err, IsNil)

		status, err := w.Status()
		c.Assert(err, IsNil)
		c.Assert(status.IsClean(), Equals, true)

		return nil
	})
}

func (s *WorktreeSuite) TestCheckoutWithGitignore(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	f, _ := fs.Create("file")
	f.Close()

	err = w.Checkout(&CheckoutOptions{})
	c.Assert(err.Error(), Equals, "worktree contains unstagged changes")

	f, _ = fs.Create(".gitignore")
	f.Write([]byte("file"))
	f.Close()

	err = w.Checkout(&CheckoutOptions{})
	c.Assert(err.Error(), Equals, "worktree contains unstagged changes")

	w.Add(".gitignore")

	err = w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)
}

func (s *WorktreeSuite) TestStatus(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	status, err := w.Status()
	c.Assert(err, IsNil)

	c.Assert(status.IsClean(), Equals, false)
	c.Assert(status, HasLen, 9)
}

func (s *WorktreeSuite) TestStatusEmpty(c *C) {
	fs := memfs.New()
	storage := memory.NewStorage()

	r, err := Init(storage, fs)
	c.Assert(err, IsNil)

	w, err := r.Worktree()
	c.Assert(err, IsNil)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status.IsClean(), Equals, true)
	c.Assert(status, NotNil)
}

func (s *WorktreeSuite) TestReset(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	commit := plumbing.NewHash("35e85108805c84807bc66a02d91535e1e24b38b9")

	err := w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	branch, err := w.r.Reference(plumbing.Master, false)
	c.Assert(err, IsNil)
	c.Assert(branch.Hash(), Not(Equals), commit)

	err = w.Reset(&ResetOptions{Commit: commit})
	c.Assert(err, IsNil)

	branch, err = w.r.Reference(plumbing.Master, false)
	c.Assert(err, IsNil)
	c.Assert(branch.Hash(), Equals, commit)
}

func (s *WorktreeSuite) TestResetMerge(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	commit := plumbing.NewHash("35e85108805c84807bc66a02d91535e1e24b38b9")

	err := w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	f, err := fs.Create(".gitignore")
	c.Assert(err, IsNil)
	_, err = f.Write([]byte("foo"))
	c.Assert(err, IsNil)
	err = f.Close()
	c.Assert(err, IsNil)

	err = w.Reset(&ResetOptions{Mode: MergeReset, Commit: commit})
	c.Assert(err, Equals, ErrUnstaggedChanges)

	branch, err := w.r.Reference(plumbing.Master, false)
	c.Assert(err, IsNil)
	c.Assert(branch.Hash(), Not(Equals), commit)
}

func (s *WorktreeSuite) TestResetHard(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	commit := plumbing.NewHash("35e85108805c84807bc66a02d91535e1e24b38b9")

	err := w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	f, err := fs.Create(".gitignore")
	c.Assert(err, IsNil)
	_, err = f.Write([]byte("foo"))
	c.Assert(err, IsNil)
	err = f.Close()
	c.Assert(err, IsNil)

	err = w.Reset(&ResetOptions{Mode: HardReset, Commit: commit})
	c.Assert(err, IsNil)

	branch, err := w.r.Reference(plumbing.Master, false)
	c.Assert(err, IsNil)
	c.Assert(branch.Hash(), Equals, commit)
}

func (s *WorktreeSuite) TestStatusAfterCheckout(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{Force: true})
	c.Assert(err, IsNil)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status.IsClean(), Equals, true)

}

func (s *WorktreeSuite) TestStatusModified(c *C) {
	dir, err := ioutil.TempDir("", "status")
	defer os.RemoveAll(dir)

	fs := osfs.New(filepath.Join(dir, "worktree"))
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err = w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	f, err := fs.Create(".gitignore")
	c.Assert(err, IsNil)
	_, err = f.Write([]byte("foo"))
	c.Assert(err, IsNil)
	err = f.Close()
	c.Assert(err, IsNil)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status.IsClean(), Equals, false)
	c.Assert(status.File(".gitignore").Worktree, Equals, Modified)
}

func (s *WorktreeSuite) TestStatusIgnored(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	w.Checkout(&CheckoutOptions{})

	fs.MkdirAll("another", os.ModePerm)
	f, _ := fs.Create("another/file")
	f.Close()
	fs.MkdirAll("vendor/github.com", os.ModePerm)
	f, _ = fs.Create("vendor/github.com/file")
	f.Close()
	fs.MkdirAll("vendor/gopkg.in", os.ModePerm)
	f, _ = fs.Create("vendor/gopkg.in/file")
	f.Close()
	f, _ = fs.Create("vendor/gopkg.in/another")
	f.Close()

	status, _ := w.Status()
	c.Assert(len(status), Equals, 4)
	_, ok := status["another/file"]
	c.Assert(ok, Equals, true)
	_, ok = status["vendor/github.com/file"]
	c.Assert(ok, Equals, true)
	_, ok = status["vendor/gopkg.in/file"]
	c.Assert(ok, Equals, true)
	_, ok = status["vendor/gopkg.in/another"]
	c.Assert(ok, Equals, true)

	f, _ = fs.Create(".gitignore")
	f.Write([]byte("vendor/g*/"))
	f.Close()
	f, _ = fs.Create("vendor/.gitignore")
	f.Write([]byte("!github.com/\n"))
	f.Close()

	status, _ = w.Status()
	c.Assert(len(status), Equals, 4)
	_, ok = status[".gitignore"]
	c.Assert(ok, Equals, true)
	_, ok = status["another/file"]
	c.Assert(ok, Equals, true)
	_, ok = status["vendor/.gitignore"]
	c.Assert(ok, Equals, true)
	_, ok = status["vendor/github.com/file"]
	c.Assert(ok, Equals, true)
}

func setForcedIgnored(v bool) {
	forceIgnored = v
}

func (s *WorktreeSuite) TestStatusForceIgnored(c *C) {
	setForcedIgnored(true)
	defer setForcedIgnored(false)

	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	w.Checkout(&CheckoutOptions{})

	fs.MkdirAll("another", os.ModePerm)
	f, _ := fs.Create("another/file")
	f.Close()
	fs.MkdirAll("vendor/github.com", os.ModePerm)
	f, _ = fs.Create("vendor/github.com/file")
	f.Close()
	fs.MkdirAll("vendor/gopkg.in", os.ModePerm)
	f, _ = fs.Create("vendor/gopkg.in/file")
	f.Close()
	f, _ = fs.Create("vendor/gopkg.in/another")
	f.Close()

	status, _ := w.Status()
	c.Assert(len(status), Equals, 4)

	f, _ = fs.Create(".gitignore")
	f.Write([]byte("vendor/g*/"))
	f.Close()
	f, _ = fs.Create("vendor/.gitignore")
	f.Write([]byte("!github.com/\n"))
	f.Close()

	status, _ = w.Status()
	c.Assert(len(status), Equals, 4)

	_, err := w.Add("vendor/gopkg.in/file")
	c.Assert(err, Equals, nil)
	status, _ = w.Status()
	c.Assert(len(status), Equals, 5)
	_, ok := status["vendor/gopkg.in/file"]
	c.Assert(ok, Equals, true)

	_, err = w.Commit("test", &CommitOptions{Author:&object.Signature{Name: "test", Email: "test@test.com", When: time.Now()}})
	c.Assert(err, Equals, nil)

	status, _ = w.Status()
	c.Assert(len(status), Equals, 4)

	_, err = w.Remove("vendor/gopkg.in/file")
	c.Assert(err, Equals, nil)

	status, _ = w.Status()
	c.Assert(len(status), Equals, 5)
	_, ok = status["vendor/gopkg.in/file"]
	c.Assert(ok, Equals, true)
}

func (s *WorktreeSuite) TestStatusUntracked(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{Force: true})
	c.Assert(err, IsNil)

	f, err := w.fs.Create("foo")
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status.File("foo").Staging, Equals, Untracked)
	c.Assert(status.File("foo").Worktree, Equals, Untracked)
}

func (s *WorktreeSuite) TestStatusDeleted(c *C) {
	dir, err := ioutil.TempDir("", "status")
	defer os.RemoveAll(dir)

	fs := osfs.New(filepath.Join(dir, "worktree"))
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err = w.Checkout(&CheckoutOptions{})
	c.Assert(err, IsNil)

	err = fs.Remove(".gitignore")
	c.Assert(err, IsNil)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status.IsClean(), Equals, false)
	c.Assert(status.File(".gitignore").Worktree, Equals, Deleted)
}

func (s *WorktreeSuite) TestSubmodule(c *C) {
	path := fixtures.ByTag("submodule").One().Worktree().Root()
	r, err := PlainOpen(path)
	c.Assert(err, IsNil)

	w, err := r.Worktree()
	c.Assert(err, IsNil)

	m, err := w.Submodule("basic")
	c.Assert(err, IsNil)

	c.Assert(m.Config().Name, Equals, "basic")
}

func (s *WorktreeSuite) TestSubmodules(c *C) {
	path := fixtures.ByTag("submodule").One().Worktree().Root()
	r, err := PlainOpen(path)
	c.Assert(err, IsNil)

	w, err := r.Worktree()
	c.Assert(err, IsNil)

	l, err := w.Submodules()
	c.Assert(err, IsNil)

	c.Assert(l, HasLen, 2)
}

func (s *WorktreeSuite) TestAddUntracked(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{Force: true})
	c.Assert(err, IsNil)

	idx, err := w.r.Storer.Index()
	c.Assert(err, IsNil)
	c.Assert(idx.Entries, HasLen, 9)

	err = util.WriteFile(w.fs, "foo", []byte("FOO"), 0755)
	c.Assert(err, IsNil)

	hash, err := w.Add("foo")
	c.Assert(hash.String(), Equals, "d96c7efbfec2814ae0301ad054dc8d9fc416c9b5")
	c.Assert(err, IsNil)

	idx, err = w.r.Storer.Index()
	c.Assert(err, IsNil)
	c.Assert(idx.Entries, HasLen, 10)

	e, err := idx.Entry("foo")
	c.Assert(err, IsNil)
	c.Assert(e.Hash, Equals, hash)
	c.Assert(e.Mode, Equals, filemode.Executable)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status, HasLen, 1)

	file := status.File("foo")
	c.Assert(file.Staging, Equals, Added)
	c.Assert(file.Worktree, Equals, Unmodified)

	obj, err := w.r.Storer.EncodedObject(plumbing.BlobObject, hash)
	c.Assert(err, IsNil)
	c.Assert(obj, NotNil)
	c.Assert(obj.Size(), Equals, int64(3))
}

func (s *WorktreeSuite) TestAddModified(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{Force: true})
	c.Assert(err, IsNil)

	idx, err := w.r.Storer.Index()
	c.Assert(err, IsNil)
	c.Assert(idx.Entries, HasLen, 9)

	err = util.WriteFile(w.fs, "LICENSE", []byte("FOO"), 0644)
	c.Assert(err, IsNil)

	hash, err := w.Add("LICENSE")
	c.Assert(err, IsNil)
	c.Assert(hash.String(), Equals, "d96c7efbfec2814ae0301ad054dc8d9fc416c9b5")

	idx, err = w.r.Storer.Index()
	c.Assert(err, IsNil)
	c.Assert(idx.Entries, HasLen, 9)

	e, err := idx.Entry("LICENSE")
	c.Assert(err, IsNil)
	c.Assert(e.Hash, Equals, hash)
	c.Assert(e.Mode, Equals, filemode.Regular)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status, HasLen, 1)

	file := status.File("LICENSE")
	c.Assert(file.Staging, Equals, Modified)
	c.Assert(file.Worktree, Equals, Unmodified)
}

func (s *WorktreeSuite) TestAddUnmodified(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{Force: true})
	c.Assert(err, IsNil)

	hash, err := w.Add("LICENSE")
	c.Assert(hash.String(), Equals, "c192bd6a24ea1ab01d78686e417c8bdc7c3d197f")
	c.Assert(err, IsNil)
}

func (s *WorktreeSuite) TestAddSymlink(c *C) {
	dir, err := ioutil.TempDir("", "checkout")
	defer os.RemoveAll(dir)

	r, err := PlainInit(dir, false)
	c.Assert(err, IsNil)
	err = util.WriteFile(r.wt, "foo", []byte("qux"), 0644)
	c.Assert(err, IsNil)
	err = r.wt.Symlink("foo", "bar")
	c.Assert(err, IsNil)

	w, err := r.Worktree()
	c.Assert(err, IsNil)
	h, err := w.Add("foo")
	c.Assert(err, IsNil)
	c.Assert(h, Not(Equals), plumbing.NewHash("19102815663d23f8b75a47e7a01965dcdc96468c"))

	h, err = w.Add("bar")
	c.Assert(err, IsNil)
	c.Assert(h, Equals, plumbing.NewHash("19102815663d23f8b75a47e7a01965dcdc96468c"))

	obj, err := w.r.Storer.EncodedObject(plumbing.BlobObject, h)
	c.Assert(err, IsNil)
	c.Assert(obj, NotNil)
	c.Assert(obj.Size(), Equals, int64(3))

}

func (s *WorktreeSuite) TestRemove(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{Force: true})
	c.Assert(err, IsNil)

	hash, err := w.Remove("LICENSE")
	c.Assert(hash.String(), Equals, "c192bd6a24ea1ab01d78686e417c8bdc7c3d197f")
	c.Assert(err, IsNil)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status, HasLen, 1)
	c.Assert(status.File("LICENSE").Staging, Equals, Deleted)
}

func (s *WorktreeSuite) TestRemoveNotExistentEntry(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{Force: true})
	c.Assert(err, IsNil)

	hash, err := w.Remove("not-exists")
	c.Assert(hash.IsZero(), Equals, true)
	c.Assert(err, NotNil)
}

func (s *WorktreeSuite) TestRemoveDeletedFromWorktree(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{Force: true})
	c.Assert(err, IsNil)

	err = fs.Remove("LICENSE")
	c.Assert(err, IsNil)

	hash, err := w.Remove("LICENSE")
	c.Assert(hash.String(), Equals, "c192bd6a24ea1ab01d78686e417c8bdc7c3d197f")
	c.Assert(err, IsNil)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status, HasLen, 1)
	c.Assert(status.File("LICENSE").Staging, Equals, Deleted)
}

func (s *WorktreeSuite) TestMove(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{Force: true})
	c.Assert(err, IsNil)

	hash, err := w.Move("LICENSE", "foo")
	c.Check(hash.String(), Equals, "c192bd6a24ea1ab01d78686e417c8bdc7c3d197f")
	c.Assert(err, IsNil)

	status, err := w.Status()
	c.Assert(err, IsNil)
	c.Assert(status, HasLen, 2)
	c.Assert(status.File("LICENSE").Staging, Equals, Deleted)
	c.Assert(status.File("foo").Staging, Equals, Added)

}

func (s *WorktreeSuite) TestMoveNotExistentEntry(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{Force: true})
	c.Assert(err, IsNil)

	hash, err := w.Move("not-exists", "foo")
	c.Assert(hash.IsZero(), Equals, true)
	c.Assert(err, NotNil)
}

func (s *WorktreeSuite) TestMoveToExistent(c *C) {
	fs := memfs.New()
	w := &Worktree{
		r:  s.Repository,
		fs: fs,
	}

	err := w.Checkout(&CheckoutOptions{Force: true})
	c.Assert(err, IsNil)

	hash, err := w.Move(".gitignore", "LICENSE")
	c.Assert(hash.IsZero(), Equals, true)
	c.Assert(err, Equals, ErrDestinationExists)
}
