package gitdir

import (
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
	"gopkg.in/src-d/go-git.v3/clients/common"
	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/utils/tgz"
)

func Test(t *testing.T) { TestingT(t) }

var initFixtures = [...]struct {
	name         string
	tgz          string
	capabilities [][2]string
	packfile     string
}{
	{
		name: "spinnaker",
		tgz:  "fixtures/spinnaker-gc.tgz",
		capabilities: [][2]string{
			{"symref", "HEAD:refs/heads/master"},
		},
		packfile: "objects/pack/pack-584416f86235cac0d54bfabbdc399fb2b09a5269.pack",
	},
}

type fixture struct {
	path         string               // repo names to paths of the extracted tgz
	capabilities *common.Capabilities // expected capabilities
	packfile     string               // path of the packfile
}

type SuiteGitDir struct {
	fixtures map[string]fixture
}

var _ = Suite(&SuiteGitDir{})

func (s *SuiteGitDir) SetUpSuite(c *C) {
	s.fixtures = make(map[string]fixture, len(initFixtures))

	for _, init := range initFixtures {
		comment := Commentf("fixture name = %s\n", init.name)

		path, err := tgz.Extract(init.tgz)
		c.Assert(err, IsNil, comment)

		fixt := fixture{}

		fixt.path = filepath.Join(path, ".git")

		fixt.capabilities = common.NewCapabilities()
		for _, pair := range init.capabilities {
			fixt.capabilities.Add(pair[0], pair[1])
		}

		fixt.packfile = init.packfile

		s.fixtures[init.name] = fixt
	}
}

func (s *SuiteGitDir) TearDownSuite(c *C) {
	for name, fixture := range s.fixtures {
		dir := filepath.Dir(fixture.path)
		err := os.RemoveAll(dir)
		c.Assert(err, IsNil, Commentf("cannot delete tmp dir for fixture %s: %s\n",
			name, dir))
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
			input: "foo",
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
				"refs/heads/master":                                 core.NewHash("409db80e56365049edb704f2ecbd449ddf64dc0d"),
				"refs/remotes/origin/HEAD":                          core.NewHash("409db80e56365049edb704f2ecbd449ddf64dc0d"),
				"refs/remotes/origin/explicit-machine-type":         core.NewHash("f262e833a215c90b703115691f03f182c1be4b91"),
				"refs/remotes/origin/fix-aws-creds-copy":            core.NewHash("871cf4d673e0d94c6eb2558bfc7a525c2bc7e538"),
				"refs/remotes/origin/kubernetes-no-gcloud":          core.NewHash("0b553b5b6fa773f3d7a38b229d9f75627c0762aa"),
				"refs/remotes/origin/lwander-patch-igor":            core.NewHash("9c987f44908bc9aa05e950347cd03228ba199630"),
				"refs/remotes/origin/master":                        core.NewHash("409db80e56365049edb704f2ecbd449ddf64dc0d"),
				"refs/remotes/origin/revert-898-codelab-script-fix": core.NewHash("426cd84d1741d0ff68bad646bc8499b1f163a893"),
				"refs/remotes/origin/terraform-aws-prototype":       core.NewHash("a34445e7d2e758a8c953fa3a357198ec09fcba88"),
				"refs/remotes/origin/typo":                          core.NewHash("86b48b962e599c096a5870cd8047778bb32a6e1e"),
				"refs/tags/v0.10.0":                                 core.NewHash("d081d66c2a76d04ff479a3431dc36e44116fde40"),
				"refs/tags/v0.11.0":                                 core.NewHash("3e349f806a0d02bf658c3544c46a0a7a9ee78673"),
				"refs/tags/v0.12.0":                                 core.NewHash("82562fa518f0a2e2187ea2604b07b67f2e7049ae"),
				"refs/tags/v0.13.0":                                 core.NewHash("48b655898fa9c72d62e8dd73b022ecbddd6e4cc2"),
				"refs/tags/v0.14.0":                                 core.NewHash("7ecc2ad58e24a5b52504985467a10c6a3bb85b9b"),
				"refs/tags/v0.15.0":                                 core.NewHash("740e3adff4c350899db7772f8f537d1d0d96ec75"),
				"refs/tags/v0.16.0":                                 core.NewHash("466ca58a3129f1b2ead117a43535ecb410d621ac"),
				"refs/tags/v0.17.0":                                 core.NewHash("48020cb7a45603d47e6041de072fe0665e47676f"),
				"refs/tags/v0.18.0":                                 core.NewHash("6fcb9036ab4d921dbdab41baf923320484a11188"),
				"refs/tags/v0.19.0":                                 core.NewHash("a2ce1f4c9d0bde4e93dfcb90a445ed069030640c"),
				"refs/tags/v0.20.0":                                 core.NewHash("974f476f0ec5a9dcc4bb005384d449f0a5122da4"),
				"refs/tags/v0.21.0":                                 core.NewHash("e08e3917f3a0487e33cd6dcef24fe03e570b73f5"),
				"refs/tags/v0.22.0":                                 core.NewHash("834612b4f181171d5e1e263b4e7e55d609ab19f5"),
				"refs/tags/v0.23.0":                                 core.NewHash("65558da39c07a6f9104651281c226981e880b49c"),
				"refs/tags/v0.24.0":                                 core.NewHash("5c97aa1f2f784e92f065055f9e79df83fac7a4aa"),
				"refs/tags/v0.25.0":                                 core.NewHash("d6e696f9d5e2dac968638665886e2300ae15709a"),
				"refs/tags/v0.26.0":                                 core.NewHash("974861702abd8388e0507cf3f348d6d3c40acef4"),
				"refs/tags/v0.27.0":                                 core.NewHash("65771ef145b3e07e130abc84fb07f0b8044fcf59"),
				"refs/tags/v0.28.0":                                 core.NewHash("5d86433d6dc4358277a5e9a834948f0822225a6d"),
				"refs/tags/v0.29.0":                                 core.NewHash("c1582497c23d81e61963841861c5aebbf10e12ab"),
				"refs/tags/v0.3.0":                                  core.NewHash("8b6002b614b454d45bafbd244b127839421f92ff"),
				"refs/tags/v0.30.0":                                 core.NewHash("b0f26484aab0afe2f342be84583213c3c64b7eb3"),
				"refs/tags/v0.31.0":                                 core.NewHash("8a2da11c9d29e3a879a068c197568c108b9e5f88"),
				"refs/tags/v0.32.0":                                 core.NewHash("5c5fc48a1506bb4609ca5588f90cf021a29a4a37"),
				"refs/tags/v0.33.0":                                 core.NewHash("d443f1f61e23411d9ac08f0fc6bbeb8e4c46ee39"),
				"refs/tags/v0.34.0":                                 core.NewHash("0168d74697d65cde65f931254c09a6bd7ff4f0d5"),
				"refs/tags/v0.35.0":                                 core.NewHash("a46303084ad9decf71a8ea9fd1529e22c6fdd2c4"),
				"refs/tags/v0.36.0":                                 core.NewHash("4da0d7bb89e85bd5f14ff36d983a0ae773473b2d"),
				"refs/tags/v0.37.0":                                 core.NewHash("85ec60477681933961c9b64c18ada93220650ac5"),
				"refs/tags/v0.4.0":                                  core.NewHash("95ee6e6c750ded1f4dc5499bad730ce3f58c6c3a"),
				"refs/tags/v0.5.0":                                  core.NewHash("0a3fb06ff80156fb153bcdcc58b5e16c2d27625c"),
				"refs/tags/v0.6.0":                                  core.NewHash("dc22e2035292ccf020c30d226f3cc2da651773f6"),
				"refs/tags/v0.7.0":                                  core.NewHash("3f36d8f1d67538afd1f089ffd0d242fc4fda736f"),
				"refs/tags/v0.8.0":                                  core.NewHash("8526c58617f68de076358873b8aa861a354b48a9"),
				"refs/tags/v0.9.0":                                  core.NewHash("776914ef8a097f5683957719c49215a5db17c2cb"),
			},
		},
	} {
		comment := Commentf("subtest %d", i)
		_, dir := s.newFixtureDir(c, test.fixture)

		refs, err := dir.Refs()
		c.Assert(err, IsNil, comment)
		c.Assert(refs, DeepEquals, test.refs, comment)
	}
}

func (s *SuiteGitDir) newFixtureDir(c *C, fixName string) (*fixture, *Dir) {
	fixture, ok := s.fixtures[fixName]
	c.Assert(ok, Equals, true)

	dir, err := New(fixture.path)
	c.Assert(err, IsNil)

	return &fixture, dir
}

func (s *SuiteGitDir) TestCapabilities(c *C) {
	for i, test := range [...]struct {
		fixture      string
		capabilities *common.Capabilities
	}{
		{
			fixture: "spinnaker",
		},
	} {
		comment := Commentf("subtest %d", i)
		fixture, dir := s.newFixtureDir(c, test.fixture)

		capabilities, err := dir.Capabilities()
		c.Assert(err, IsNil, comment)
		c.Assert(capabilities, DeepEquals, fixture.capabilities, comment)
	}
}

func (s *SuiteGitDir) TestPackfile(c *C) {
	for i, test := range [...]struct {
		fixture      string
		capabilities *common.Capabilities
	}{
		{
			fixture: "spinnaker",
		},
	} {
		comment := Commentf("subtest %d", i)
		fixture, dir := s.newFixtureDir(c, test.fixture)

		packfile, err := dir.Packfile()
		c.Assert(err, IsNil, comment)

		relativeFixturePackfile, err := filepath.Rel(dir.path, packfile)
		c.Assert(err, IsNil, comment)

		c.Assert(relativeFixturePackfile, Equals, fixture.packfile)
	}
}
