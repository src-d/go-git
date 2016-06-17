package tgz

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteTGZ struct{}

var _ = Suite(&SuiteTGZ{})

func (s *SuiteTGZ) TestExtract(c *C) {
	for i, test := range tests {
		com := Commentf("%d) tgz path = %s", i, test.tgz)

		path, err := Extract(test.tgz)
		if test.err != "" {
			c.Assert(err, ErrorMatches, test.err, com)
		} else {
			c.Assert(err, IsNil, com)

			obt, err := relativeTree(path)
			c.Assert(err, IsNil, com)

			sort.Strings(test.tree)
			c.Assert(obt, DeepEquals, test.tree, com)

			err = os.RemoveAll(path)
			c.Assert(err, IsNil, com)
		}
	}
}

var tests = [...]struct {
	tgz  string
	tree []string
	err  string // error regexp to match
}{
	{
		tgz: "not-found",
		err: "open not-found: no such file .*",
	}, {
		tgz: "fixtures/invalid-gzip.tgz",
		err: "gzip: invalid header",
	}, {
		tgz: "fixtures/not-a-tar.tgz",
		err: "unexpected EOF",
	}, {
		tgz: "fixtures/test-01.tgz",
		tree: []string{
			"foo.txt",
		},
	}, {
		tgz: "fixtures/test-02.tgz",
		tree: []string{
			"baz.txt",
			"bla.txt",
			"foo.txt",
		},
	}, {
		tgz: "fixtures/test-03.tgz",
		tree: []string{
			"bar",
			"bar/baz.txt",
			"bar/foo.txt",
			"baz",
			"baz/bar",
			"baz/bar/foo.txt",
			"baz/baz",
			"baz/baz/baz",
			"baz/baz/baz/foo.txt",
			"foo.txt",
		},
	},
}

func relativeTree(path string) ([]string, error) {
	path = filepath.Clean(path)

	absPaths := []string{}
	walkFn := func(path string, info os.FileInfo, err error) error {
		absPaths = append(absPaths, path)
		return nil
	}

	_ = filepath.Walk(path, walkFn)

	return toRelative(absPaths[1:], path) // strip the base dir
}

func toRelative(paths []string, base string) ([]string, error) {
	r := []string{}
	for _, p := range paths {
		rel, err := filepath.Rel(base, p)
		if err != nil {
			return nil, err
		}
		r = append(r, rel)
	}

	return r, nil
}
