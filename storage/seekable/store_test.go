package seekable_test

import (
	"os"
	"sort"
	"strings"
	"testing"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/packfile"
	"gopkg.in/src-d/go-git.v3/storage/memory"
	"gopkg.in/src-d/go-git.v3/storage/seekable"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SeekableSuite struct{}

var _ = Suite(&SeekableSuite{})

func (s *SeekableSuite) TestNewFailNoData(c *C) {
	_, err := seekable.New("", "")
	c.Assert(err, ErrorMatches, ".* no such file or directory")
}

func (s *SeekableSuite) TestGetCompareWithMemoryStorage(c *C) {
	for i, packfilePath := range [...]string{
		"../../formats/packfile/fixtures/spinnaker-spinnaker.pack",
		"../../formats/packfile/fixtures/alcortesm-binary-relations.pack",
		"../../formats/packfile/fixtures/git-fixture.ref-delta",
	} {
		com := Commentf("at subtest %d", i)

		memSto := memory.NewObjectStorage()
		f, err := os.Open(packfilePath)
		c.Assert(err, IsNil, com)

		d := packfile.NewDecoder(f)
		_, err = d.Decode(memSto)
		c.Assert(err, IsNil, com)

		err = f.Close()
		c.Assert(err, IsNil, com)

		lastDot := strings.LastIndex(packfilePath, ".")
		idxPath := packfilePath[:lastDot] + ".idx"

		sto, err := seekable.New(packfilePath, idxPath)
		c.Assert(err, IsNil, com)

		for _, typ := range [...]core.ObjectType{
			core.CommitObject,
			core.TreeObject,
			core.BlobObject,
			core.TagObject,
		} {
			iter, err := memSto.Iter(typ)
			c.Assert(err, IsNil, com)

			for {
				memObject, err := iter.Next()
				if err != nil {
					iter.Close()
					break
				}

				obt, err := sto.Get(memObject.Hash())
				c.Assert(err, IsNil, com)

				c.Assert(obt.Type(), Equals, memObject.Type(), com)
				c.Assert(obt.Size(), Equals, memObject.Size(), com)
				if memObject.Content() != nil {
					c.Assert(obt.Content(), DeepEquals, memObject.Content(),
						com)
				}
				c.Assert(obt.Hash(), Equals, memObject.Hash(), com)
			}

			iter.Close()
		}
	}
}

func (s *SeekableSuite) TestIterCompareWithMemoryStorage(c *C) {
	for i, packfilePath := range [...]string{
		"../../formats/packfile/fixtures/spinnaker-spinnaker.pack",
		"../../formats/packfile/fixtures/alcortesm-binary-relations.pack",
		"../../formats/packfile/fixtures/git-fixture.ref-delta",
	} {
		com := Commentf("at subtest %d", i)

		memSto := memory.NewObjectStorage()
		f, err := os.Open(packfilePath)
		c.Assert(err, IsNil, com)
		d := packfile.NewDecoder(f)
		_, err = d.Decode(memSto)
		c.Assert(err, IsNil, com)
		err = f.Close()
		c.Assert(err, IsNil, com)

		lastDot := strings.LastIndex(packfilePath, ".")
		idxPath := packfilePath[:lastDot] + ".idx"

		sto, err := seekable.New(packfilePath, idxPath)
		c.Assert(err, IsNil, com)

		for _, typ := range [...]core.ObjectType{
			core.CommitObject,
			core.TreeObject,
			core.BlobObject,
			core.TagObject,
		} {

			memObjs, err := iterToSortedSlice(memSto, typ)
			c.Assert(err, IsNil, com)

			seekableObjs, err := iterToSortedSlice(sto, typ)
			c.Assert(err, IsNil, com)

			for i, exp := range memObjs {
				c.Assert(seekableObjs[i].Hash(), Equals, exp.Hash(), com)
			}
		}
	}
}

func iterToSortedSlice(storage core.ObjectStorage, typ core.ObjectType) ([]core.Object,
	error) {

	iter, err := storage.Iter(typ)
	if err != nil {
		return nil, err
	}

	r := make([]core.Object, 0)
	for {
		obj, err := iter.Next()
		if err != nil {
			iter.Close()
			break
		}
		r = append(r, obj)
	}

	iter.Close()

	sort.Sort(byHash(r))

	return r, nil
}

type byHash []core.Object

func (a byHash) Len() int      { return len(a) }
func (a byHash) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byHash) Less(i, j int) bool {
	return a[i].Hash().String() < a[j].Hash().String()
}

func (s *SeekableSuite) TestSet(c *C) {
	path := "../../formats/packfile/fixtures/spinnaker-spinnaker.pack"
	lastDot := strings.LastIndex(path, ".")
	idxPath := path[:lastDot] + ".idx"

	sto, err := seekable.New(path, idxPath)
	c.Assert(err, IsNil)

	_, err = sto.Set(&memory.Object{})
	c.Assert(err, ErrorMatches, "set operation not permitted")
}
