package gitdir

import (
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/utils/tgz"
)

func Test(t *testing.T) { TestingT(t) }

var fixtures = [...]struct {
	name string
	tgz  string
}{
	{
		name: "spinnaker",
		tgz:  "fixtures/spinnaker-gc.tgz",
	},
}

type SuiteGitDir struct {
	fixturePath map[string]string // repo names to paths of the extracted tgz
}

var _ = Suite(&SuiteGitDir{})

func (s *SuiteGitDir) SetUpSuite(c *C) {
	s.fixturePath = make(map[string]string, len(fixtures))

	for _, fixture := range fixtures {
		comment := Commentf("fixture name = %s\n", fixture.name)

		file, err := os.Open(fixture.tgz)
		c.Assert(err, IsNil, comment)

		path, err := tgz.Extract(file)
		c.Assert(err, IsNil, comment)

		s.fixturePath[fixture.name] = filepath.Join(path, ".git")

		err = file.Close()
		c.Assert(err, IsNil, comment)
	}
}

func (s *SuiteGitDir) TearDownSuite(c *C) {
	for name, path := range s.fixturePath {
		err := os.RemoveAll(path)
		c.Assert(err, IsNil, Commentf("cannot delete tmp dir for fixture %s: %s\n",
			name, path))
	}
}

func (s *SuiteGitDir) TestNewDir(c *C) {
	for i, test := range [...]struct {
		input string
		err   error
		path  string
	}{
		{
			input: "",
			err:   ErrBadGitDirName,
		}, {
			input: "/",
			err:   ErrBadGitDirName,
		}, {
			input: "/tmp/foo",
			err:   ErrBadGitDirName,
		}, {
			input: "/tmp/../tmp/foo/.git",
			path:  "/tmp/foo/.git",
		},
	} {
		comment := Commentf("subtest %d", i)

		dir, err := New(test.input)
		c.Assert(err, Equals, test.err, comment)
		if test.err == nil {
			c.Assert(dir.path, Equals, test.path, comment)
		}
	}
}

func (s *SuiteGitDir) TestRefs(c *C) {
	for i, test := range [...]struct {
		fixture string
		refs    map[string]core.Hash
	}{
		{
			fixture: "spinnaker",
			refs: map[string]core.Hash{
				"/refs/bla": core.NewHash("fasdfasd"),
			},
		},
	} {
		comment := Commentf("subtest %d", i)

		dir, err := New(s.fixturePath[test.fixture])
		c.Assert(err, IsNil, comment)

		refs, err := dir.Refs()
		c.Assert(err, IsNil, comment)
		c.Assert(refs, DeepEquals, test.refs, comment)
	}
}
