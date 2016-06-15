package fs_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/packfile"
	"gopkg.in/src-d/go-git.v3/storage/fs"
	"gopkg.in/src-d/go-git.v3/storage/fs/internal/gitdir"
	"gopkg.in/src-d/go-git.v3/storage/memory"
	"gopkg.in/src-d/go-git.v3/utils/tgz"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SeekableSuite struct{}

var _ = Suite(&SeekableSuite{})

func (s *SeekableSuite) TestNewFailNoData(c *C) {
	_, err := fs.New("not_found/.git")
	c.Assert(err, Equals, gitdir.ErrNotFound)

	_, err = fs.New("not_found")
	c.Assert(err, Equals, gitdir.ErrBadGitDirName)
}

func (s *SeekableSuite) TestGetCompareWithMemoryStorage(c *C) {
	for i, tgzPath := range [...]string{
		"internal/gitdir/fixtures/spinnaker-gc.tgz",
	} {
		com := Commentf("at subtest %d, (tgz = %q)", i, tgzPath)

		path, err := tgz.Extract(tgzPath)
		c.Assert(err, IsNil, com)
		com = Commentf("at subtest %d, (tgz = %q, extracted to %q)",
			i, tgzPath, path)
		path = path + "/.git"

		memSto, err := memStorageFromGitDir(path)
		c.Assert(err, IsNil, com)

		fsSto, err := fs.New(path)
		c.Assert(err, IsNil, com)

		equal, reason, err := equalsStorages(memSto, fsSto)
		c.Assert(err, IsNil, com)
		c.Assert(equal, Equals, true,
			Commentf("%s - %s\n", com.CheckCommentString(), reason))

		err = os.RemoveAll(path)
		c.Assert(err, IsNil, com)
	}
}

func memStorageFromGitDir(path string) (*memory.ObjectStorage, error) {
	dir, err := gitdir.New(path)
	if err != nil {
		return nil, err
	}

	fmt.Println(path)
	packfilePath, err := dir.Packfile()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(packfilePath)
	if err != nil {
		return nil, err
	}

	sto := memory.NewObjectStorage()
	d := packfile.NewDecoder(f)
	_, err = d.Decode(sto)
	if err != nil {
		return nil, err
	}

	err = f.Close()
	if err != nil {
		return nil, err
	}

	return sto, nil
}

func equalsStorages(a, b core.ObjectStorage) (bool, string, error) {
	for _, typ := range [...]core.ObjectType{
		core.CommitObject,
		core.TreeObject,
		core.BlobObject,
		core.TagObject,
	} {
		iter, err := a.Iter(typ)
		if err != nil {
			return false, "", err
		}

		for {
			ao, err := iter.Next()
			if err != nil {
				iter.Close()
				break
			}

			bo, err := b.Get(ao.Hash())
			if err != nil {
				return false, "", err
			}

			equal, reason, err := equalsObjects(ao, bo)
			if !equal || err != nil {
				return equal, reason, err
			}
		}

		iter.Close()
	}

	return true, "", nil
}

func equalsObjects(a, b core.Object) (bool, string, error) {
	ah := a.Hash()
	bh := b.Hash()
	if ah != bh {
		return false, fmt.Sprintf("object hashes differ: %s and %s\n",
			ah, bh), nil
	}

	atyp := a.Type()
	btyp := b.Type()
	if atyp != btyp {
		return false, fmt.Sprintf("object types differ: %d and %d\n",
			atyp, btyp), nil
	}

	asz := a.Size()
	bsz := b.Size()
	if asz != bsz {
		return false, fmt.Sprintf("object sizes differ: %d and %d\n",
			asz, bsz), nil
	}

	ac := a.Content()
	if ac != nil {
		bc := b.Content()
		if !reflect.DeepEqual(ac, bc) {
			return false, fmt.Sprintf("object contents differ"), nil
		}
	}

	return true, "", nil
}

/*
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

		sto, err := fs.New(packfilePath, idxPath)
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

		sto, err := fs.New(packfilePath, idxPath)
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

	sto, err := fs.New(path, idxPath)
	c.Assert(err, IsNil)

	_, err = sto.Set(&memory.Object{})
	c.Assert(err, ErrorMatches, "set operation not permitted")
}
*/
