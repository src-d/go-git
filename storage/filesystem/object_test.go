package filesystem

import (
	"io/ioutil"
	"testing"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/filesystem/dotgit"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git-fixtures.v3"
)

type FsSuite struct {
	fixtures.Suite
}

var objectTypes = []plumbing.ObjectType{
	plumbing.CommitObject,
	plumbing.TagObject,
	plumbing.TreeObject,
	plumbing.BlobObject,
}

var _ = Suite(&FsSuite{})

func (s *FsSuite) TestGetFromObjectFile(c *C) {
	fs := fixtures.ByTag(".git").ByTag("unpacked").One().DotGit()
	o := NewObjectStorage(dotgit.New(fs))

	expected := plumbing.NewHash("f3dfe29d268303fc6e1bbce268605fc99573406e")
	obj, err := o.EncodedObject(plumbing.AnyObject, expected)
	c.Assert(err, IsNil)
	c.Assert(obj.Hash(), Equals, expected)
}

func (s *FsSuite) TestGetFromPackfile(c *C) {
	fixtures.Basic().ByTag(".git").Test(c, func(f *fixtures.Fixture) {
		fs := f.DotGit()
		o := NewObjectStorage(dotgit.New(fs))

		expected := plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
		obj, err := o.EncodedObject(plumbing.AnyObject, expected)
		c.Assert(err, IsNil)
		c.Assert(obj.Hash(), Equals, expected)
	})
}

func (s *FsSuite) TestGetFromPackfileMultiplePackfiles(c *C) {
	fs := fixtures.ByTag(".git").ByTag("multi-packfile").One().DotGit()
	o := NewObjectStorage(dotgit.New(fs))

	expected := plumbing.NewHash("8d45a34641d73851e01d3754320b33bb5be3c4d3")
	obj, err := o.getFromPackfile(expected, false)
	c.Assert(err, IsNil)
	c.Assert(obj.Hash(), Equals, expected)

	expected = plumbing.NewHash("e9cfa4c9ca160546efd7e8582ec77952a27b17db")
	obj, err = o.getFromPackfile(expected, false)
	c.Assert(err, IsNil)
	c.Assert(obj.Hash(), Equals, expected)
}

func (s *FsSuite) TestIter(c *C) {
	fixtures.ByTag(".git").ByTag("packfile").Test(c, func(f *fixtures.Fixture) {
		fs := f.DotGit()
		o := NewObjectStorage(dotgit.New(fs))

		iter, err := o.IterEncodedObjects(plumbing.AnyObject)
		c.Assert(err, IsNil)

		var count int32
		err = iter.ForEach(func(o plumbing.EncodedObject) error {
			count++
			return nil
		})

		c.Assert(err, IsNil)
		c.Assert(count, Equals, f.ObjectsCount)
	})
}

func (s *FsSuite) TestIterWithType(c *C) {
	fixtures.ByTag(".git").Test(c, func(f *fixtures.Fixture) {
		for _, t := range objectTypes {
			fs := f.DotGit()
			o := NewObjectStorage(dotgit.New(fs))

			iter, err := o.IterEncodedObjects(t)
			c.Assert(err, IsNil)

			err = iter.ForEach(func(o plumbing.EncodedObject) error {
				c.Assert(o.Type(), Equals, t)
				return nil
			})

			c.Assert(err, IsNil)
		}

	})
}

func (s *FsSuite) TestPackfileIter(c *C) {
	fixtures.ByTag(".git").Test(c, func(f *fixtures.Fixture) {
		fs := f.DotGit()
		dg := dotgit.New(fs)

		for _, t := range objectTypes {
			ph, err := dg.ObjectPacks()
			c.Assert(err, IsNil)

			for _, h := range ph {
				f, err := dg.ObjectPack(h)
				c.Assert(err, IsNil)

				idxf, err := dg.ObjectPackIdx(h)
				c.Assert(err, IsNil)

				iter, err := NewPackfileIter(fs, f, idxf, t)
				c.Assert(err, IsNil)
				err = iter.ForEach(func(o plumbing.EncodedObject) error {
					c.Assert(o.Type(), Equals, t)
					return nil
				})

				c.Assert(err, IsNil)
			}
		}
	})

}

func BenchmarkPackfileIter(b *testing.B) {
	if err := fixtures.Init(); err != nil {
		b.Fatal(err)
	}

	defer func() {
		if err := fixtures.Clean(); err != nil {
			b.Fatal(err)
		}
	}()

	for _, f := range fixtures.ByTag(".git") {
		b.Run(f.URL, func(b *testing.B) {
			fs := f.DotGit()
			dg := dotgit.New(fs)

			for i := 0; i < b.N; i++ {
				for _, t := range objectTypes {
					ph, err := dg.ObjectPacks()
					if err != nil {
						b.Fatal(err)
					}

					for _, h := range ph {
						f, err := dg.ObjectPack(h)
						if err != nil {
							b.Fatal(err)
						}

						idxf, err := dg.ObjectPackIdx(h)
						if err != nil {
							b.Fatal(err)
						}

						iter, err := NewPackfileIter(fs, f, idxf, t)
						if err != nil {
							b.Fatal(err)
						}

						err = iter.ForEach(func(o plumbing.EncodedObject) error {
							if o.Type() != t {
								b.Errorf("expecting %s, got %s", t, o.Type())
							}
							return nil
						})

						if err != nil {
							b.Fatal(err)
						}
					}
				}
			}
		})
	}
}

func BenchmarkPackfileIterReadContent(b *testing.B) {
	if err := fixtures.Init(); err != nil {
		b.Fatal(err)
	}

	defer func() {
		if err := fixtures.Clean(); err != nil {
			b.Fatal(err)
		}
	}()

	for _, f := range fixtures.ByTag(".git") {
		b.Run(f.URL, func(b *testing.B) {
			fs := f.DotGit()
			dg := dotgit.New(fs)

			for i := 0; i < b.N; i++ {
				for _, t := range objectTypes {
					ph, err := dg.ObjectPacks()
					if err != nil {
						b.Fatal(err)
					}

					for _, h := range ph {
						f, err := dg.ObjectPack(h)
						if err != nil {
							b.Fatal(err)
						}

						idxf, err := dg.ObjectPackIdx(h)
						if err != nil {
							b.Fatal(err)
						}

						iter, err := NewPackfileIter(fs, f, idxf, t)
						if err != nil {
							b.Fatal(err)
						}

						err = iter.ForEach(func(o plumbing.EncodedObject) error {
							if o.Type() != t {
								b.Errorf("expecting %s, got %s", t, o.Type())
							}

							r, err := o.Reader()
							if err != nil {
								b.Fatal(err)
							}

							if _, err := ioutil.ReadAll(r); err != nil {
								b.Fatal(err)
							}

							return r.Close()
						})

						if err != nil {
							b.Fatal(err)
						}
					}
				}
			}
		})
	}
}

func BenchmarkGetObjectFromPackfile(b *testing.B) {
	if err := fixtures.Init(); err != nil {
		b.Fatal(err)
	}

	defer func() {
		if err := fixtures.Clean(); err != nil {
			b.Fatal(err)
		}
	}()

	for _, f := range fixtures.Basic() {
		b.Run(f.URL, func(b *testing.B) {
			fs := f.DotGit()
			o := NewObjectStorage(dotgit.New(fs))

			for i := 0; i < b.N; i++ {
				expected := plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
				obj, err := o.EncodedObject(plumbing.AnyObject, expected)
				if err != nil {
					b.Fatal(err)
				}

				if obj.Hash() != expected {
					b.Errorf("expecting %s, got %s", expected, obj.Hash())
				}
			}
		})
	}
}
