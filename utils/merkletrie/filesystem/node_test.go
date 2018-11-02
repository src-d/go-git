package filesystem

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path"
	"testing"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie/noder"
)

func Test(t *testing.T) { TestingT(t) }

type NoderSuite struct{}

var _ = Suite(&NoderSuite{})

func (s *NoderSuite) TestDiff(c *C) {
	fsA := memfs.New()
	WriteFile(fsA, "foo", []byte("foo"), 0644)
	WriteFile(fsA, "qux/bar", []byte("foo"), 0644)
	WriteFile(fsA, "qux/qux", []byte("foo"), 0644)
	fsA.Symlink("foo", "bar")

	fsB := memfs.New()
	WriteFile(fsB, "foo", []byte("foo"), 0644)
	WriteFile(fsB, "qux/bar", []byte("foo"), 0644)
	WriteFile(fsB, "qux/qux", []byte("foo"), 0644)
	fsB.Symlink("foo", "bar")

	ch, err := merkletrie.DiffTree(
		NewRootNode(fsA, nil),
		NewRootNode(fsB, nil),
		IsEquals,
	)

	c.Assert(err, IsNil)
	c.Assert(ch, HasLen, 0)
}

func (s *NoderSuite) TestDiffChangeLink(c *C) {
	fsA := memfs.New()
	fsA.Symlink("qux", "foo")

	fsB := memfs.New()
	fsB.Symlink("bar", "foo")

	ch, err := merkletrie.DiffTree(
		NewRootNode(fsA, nil),
		NewRootNode(fsB, nil),
		IsEquals,
	)

	c.Assert(err, IsNil)
	c.Assert(ch, HasLen, 1)
}

func (s *NoderSuite) TestDiffChangeContent(c *C) {
	fsA := memfs.New()
	WriteFile(fsA, "foo", []byte("foo"), 0644)
	WriteFile(fsA, "qux/bar", []byte("foo"), 0644)
	WriteFile(fsA, "qux/qux", []byte("foo"), 0644)

	fsB := memfs.New()
	WriteFile(fsB, "foo", []byte("foo"), 0644)
	WriteFile(fsB, "qux/bar", []byte("bar"), 0644)
	WriteFile(fsB, "qux/qux", []byte("foo"), 0644)

	ch, err := merkletrie.DiffTree(
		NewRootNode(fsA, nil),
		NewRootNode(fsB, nil),
		IsEquals,
	)

	c.Assert(err, IsNil)
	c.Assert(ch, HasLen, 1)
}

func (s *NoderSuite) TestDiffSymlinkDirOnA(c *C) {
	fsA := memfs.New()
	WriteFile(fsA, "qux/qux", []byte("foo"), 0644)

	fsB := memfs.New()
	fsB.Symlink("qux", "foo")
	WriteFile(fsB, "qux/qux", []byte("foo"), 0644)

	ch, err := merkletrie.DiffTree(
		NewRootNode(fsA, nil),
		NewRootNode(fsB, nil),
		IsEquals,
	)

	c.Assert(err, IsNil)
	c.Assert(ch, HasLen, 1)
}

func (s *NoderSuite) TestDiffSymlinkDirOnB(c *C) {
	fsA := memfs.New()
	fsA.Symlink("qux", "foo")
	WriteFile(fsA, "qux/qux", []byte("foo"), 0644)

	fsB := memfs.New()
	WriteFile(fsB, "qux/qux", []byte("foo"), 0644)

	ch, err := merkletrie.DiffTree(
		NewRootNode(fsA, nil),
		NewRootNode(fsB, nil),
		IsEquals,
	)

	c.Assert(err, IsNil)
	c.Assert(ch, HasLen, 1)
}

func (s *NoderSuite) TestDiffChangeMissing(c *C) {
	fsA := memfs.New()
	WriteFile(fsA, "foo", []byte("foo"), 0644)

	fsB := memfs.New()
	WriteFile(fsB, "bar", []byte("bar"), 0644)

	ch, err := merkletrie.DiffTree(
		NewRootNode(fsA, nil),
		NewRootNode(fsB, nil),
		IsEquals,
	)

	c.Assert(err, IsNil)
	c.Assert(ch, HasLen, 2)
}

func (s *NoderSuite) TestDiffChangeMode(c *C) {
	fsA := memfs.New()
	WriteFile(fsA, "foo", []byte("foo"), 0644)

	fsB := memfs.New()
	WriteFile(fsB, "foo", []byte("foo"), 0755)

	ch, err := merkletrie.DiffTree(
		NewRootNode(fsA, nil),
		NewRootNode(fsB, nil),
		IsEquals,
	)

	c.Assert(err, IsNil)
	c.Assert(ch, HasLen, 1)
}

func (s *NoderSuite) TestDiffChangeModeNotRelevant(c *C) {
	fsA := memfs.New()
	WriteFile(fsA, "foo", []byte("foo"), 0644)

	fsB := memfs.New()
	WriteFile(fsB, "foo", []byte("foo"), 0655)

	ch, err := merkletrie.DiffTree(
		NewRootNode(fsA, nil),
		NewRootNode(fsB, nil),
		IsEquals,
	)

	c.Assert(err, IsNil)
	c.Assert(ch, HasLen, 0)
}

func (s *NoderSuite) TestDiffDirectory(c *C) {
	dir := path.Join("qux", "bar")
	fsA := memfs.New()
	fsA.MkdirAll(dir, 0644)

	fsB := memfs.New()
	fsB.MkdirAll(dir, 0644)

	ch, err := merkletrie.DiffTree(
		NewRootNode(fsA, map[string]plumbing.Hash{
			dir: plumbing.NewHash("aa102815663d23f8b75a47e7a01965dcdc96468c"),
		}),
		NewRootNode(fsB, map[string]plumbing.Hash{
			dir: plumbing.NewHash("19102815663d23f8b75a47e7a01965dcdc96468c"),
		}),
		IsEquals,
	)

	c.Assert(err, IsNil)
	c.Assert(ch, HasLen, 1)

	a, err := ch[0].Action()
	c.Assert(err, IsNil)
	c.Assert(a, Equals, merkletrie.Modify)
}

func (s *NoderSuite) TestSymlink(c *C) {
	fs := memfs.New()
	vimPath := "/usr/bin/vim"
	fs.Symlink(vimPath, "vi")

	n := NewRootNode(fs, nil)
	cs, err := n.Children()
	c.Assert(err, IsNil)
	c.Assert(cs, HasLen, 1)
	vi := cs[0]
	h := vi.Hash()

	hasher := sha1.New()
	_, err = hasher.Write([]byte(fmt.Sprintf("blob %d\x00%s", len(vimPath), vimPath)))
	c.Assert(err, IsNil)
	expected := hasher.Sum(nil)

	c.Assert(expected, DeepEquals, []byte{0xa9, 0x1b, 0xec, 0x4d, 0xac, 0x36, 0x58, 0x9a, 0x97, 0x85, 0x8b, 0x1b, 0x4b, 0x0b, 0x42, 0x90, 0x49, 0x2d, 0xf4, 0xdb})
	c.Assert(h, DeepEquals, expected)
}

func WriteFile(fs billy.Filesystem, filename string, data []byte, perm os.FileMode) error {
	f, err := fs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

var empty = make([]byte, 24)

func IsEquals(a, b noder.Hasher) bool {
	if bytes.Equal(a.Hash(), empty) || bytes.Equal(b.Hash(), empty) {
		return false
	}

	return bytes.Equal(a.Hash(), b.Hash())
}
