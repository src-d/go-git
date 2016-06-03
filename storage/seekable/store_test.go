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
	_, err := seekable.New("", nil)
	c.Assert(err, ErrorMatches, ".* no such file or directory")
}

func (s *SeekableSuite) TestGetCompareWithMemoryStorage(c *C) {
	for i, packfilePath := range [...]string{
		"../../formats/packfile/fixtures/spinnaker-spinnaker.pack",
		"../../formats/packfile/fixtures/alcortesm-binary-relations.pack",
		"../../formats/packfile/fixtures/git-fixture.ref-delta",
	} {
		comment := Commentf("at subtest %d", i)

		memStorage := memory.NewObjectStorage()
		packfileFile, err := os.Open(packfilePath)
		c.Assert(err, IsNil, comment)
		pr := packfile.NewReader(packfileFile)
		_, err = pr.Read(memStorage)
		c.Assert(err, IsNil, comment)
		err = packfileFile.Close()
		c.Assert(err, IsNil, comment)

		lastDot := strings.LastIndex(packfilePath, ".")
		idxPath := packfilePath[:lastDot] + ".idx"
		idx, err := os.Open(idxPath)
		c.Assert(err, IsNil, comment)

		storage, err := seekable.New(packfilePath, idx)
		c.Assert(err, IsNil, comment)
		err = idx.Close()
		c.Assert(err, IsNil, comment)

		for _, typ := range [...]core.ObjectType{
			core.CommitObject,
			core.TreeObject,
			core.BlobObject,
			core.TagObject,
		} {
			iter, err := memStorage.Iter(typ)
			c.Assert(err, IsNil, comment)

			for {
				memObject, err := iter.Next()
				if err != nil {
					iter.Close()
					break
				}

				obtained, err := storage.Get(memObject.Hash())
				c.Assert(err, IsNil, comment)

				c.Assert(obtained.Type(), Equals, memObject.Type(), comment)
				c.Assert(obtained.Size(), Equals, memObject.Size(), comment)
				if memObject.Content() != nil {
					c.Assert(obtained.Content(), DeepEquals, memObject.Content(),
						comment)
				}
				c.Assert(obtained.Hash(), Equals, memObject.Hash(), comment)
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
		comment := Commentf("at subtest %d", i)

		memStorage := memory.NewObjectStorage()
		packfileFile, err := os.Open(packfilePath)
		c.Assert(err, IsNil, comment)
		pr := packfile.NewReader(packfileFile)
		_, err = pr.Read(memStorage)
		c.Assert(err, IsNil, comment)
		err = packfileFile.Close()
		c.Assert(err, IsNil, comment)

		lastDot := strings.LastIndex(packfilePath, ".")
		idxPath := packfilePath[:lastDot] + ".idx"
		idx, err := os.Open(idxPath)
		c.Assert(err, IsNil, comment)

		storage, err := seekable.New(packfilePath, idx)
		c.Assert(err, IsNil, comment)
		err = idx.Close()
		c.Assert(err, IsNil, comment)

		for _, typ := range [...]core.ObjectType{
			core.CommitObject,
			core.TreeObject,
			core.BlobObject,
			core.TagObject,
		} {

			memObjects, err := iterToSortedSlice(memStorage, typ)
			c.Assert(err, IsNil, comment)

			seekableObjects, err := iterToSortedSlice(storage, typ)
			c.Assert(err, IsNil, comment)

			for i, expected := range memObjects {
				c.Assert(seekableObjects[i].Hash(), Equals, expected.Hash(), comment)
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

	result := make([]core.Object, 0)
	for {
		object, err := iter.Next()
		if err != nil {
			iter.Close()
			break
		}
		result = append(result, object)
	}

	iter.Close()

	sort.Sort(byHash(result))

	return result, nil
}

type byHash []core.Object

func (a byHash) Len() int      { return len(a) }
func (a byHash) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byHash) Less(i, j int) bool {
	return a[i].Hash().String() < a[j].Hash().String()
}
