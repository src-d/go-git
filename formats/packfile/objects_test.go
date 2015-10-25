package packfile

import (
	"encoding/base64"
	"time"

	. "gopkg.in/check.v1"
)

type ObjectsSuite struct{}

var _ = Suite(&ObjectsSuite{})

func (s *ObjectsSuite) TestComputeHash(c *C) {
	hash := ComputeHash(BlobObject, []byte(""))
	c.Assert(hash.String(), Equals, "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391")

	hash = ComputeHash(BlobObject, []byte("Hello, World!\n"))
	c.Assert(hash.String(), Equals, "8ab686eafeb1f44702738c8b0f24f2567c36da6d")
}

func (s *ObjectsSuite) TestNewHash(c *C) {
	hash := ComputeHash(BlobObject, []byte("Hello, World!\n"))

	c.Assert(hash, Equals, NewHash(hash.String()))
}

var CommitFixture = "dHJlZSBjMmQzMGZhOGVmMjg4NjE4ZjY1ZjZlZWQ2ZTE2OGUwZDUxNDg4NmY0CnBhcmVudCBiMDI5NTE3ZjYzMDBjMmRhMGY0YjY1MWI4NjQyNTA2Y2Q2YWFmNDVkCnBhcmVudCBiOGU0NzFmNThiY2JjYTYzYjA3YmRhMjBlNDI4MTkwNDA5YzJkYjQ3CmF1dGhvciBNw6F4aW1vIEN1YWRyb3MgPG1jdWFkcm9zQGdtYWlsLmNvbT4gMTQyNzgwMjQzNCArMDIwMApjb21taXR0ZXIgTcOheGltbyBDdWFkcm9zIDxtY3VhZHJvc0BnbWFpbC5jb20+IDE0Mjc4MDI0MzQgKzAyMDAKCk1lcmdlIHB1bGwgcmVxdWVzdCAjMSBmcm9tIGRyaXBvbGxlcy9mZWF0dXJlCgpDcmVhdGluZyBjaGFuZ2Vsb2c="

func (s *ObjectsSuite) TestParseCommit(c *C) {
	data, _ := base64.StdEncoding.DecodeString(CommitFixture)
	commit, err := ParseCommit(data)
	c.Assert(err, IsNil)

	c.Assert(commit.Tree.String(), Equals, "c2d30fa8ef288618f65f6eed6e168e0d514886f4")
	c.Assert(commit.Parents, HasLen, 2)
	c.Assert(commit.Parents[0].String(), Equals, "b029517f6300c2da0f4b651b8642506cd6aaf45d")
	c.Assert(commit.Parents[1].String(), Equals, "b8e471f58bcbca63b07bda20e428190409c2db47")
	c.Assert(commit.Author.Email, Equals, "mcuadros@gmail.com")
	c.Assert(commit.Author.Name, Equals, "Máximo Cuadros")
	c.Assert(commit.Author.When.Unix(), Equals, int64(1427802434))
	c.Assert(commit.Committer.Email, Equals, "mcuadros@gmail.com")
	c.Assert(commit.Message, Equals, "Merge pull request #1 from dripolles/feature\n\nCreating changelog")
}

func (s *ObjectsSuite) TestCommitHash(c *C) {
	data, _ := base64.StdEncoding.DecodeString(CommitFixture)
	commit, err := ParseCommit(data)

	c.Assert(err, IsNil)
	c.Assert(commit.Hash().String(), Equals, "a5b8b09e2f8fcb0bb99d3ccb0958157b40890d69")
}

var TreeFixture = "MTAwNjQ0IC5naXRpZ25vcmUAMoWKrTw4PtH/Cg+b3yMdVKAMnogxMDA2NDQgQ0hBTkdFTE9HANP/U+BWSp+H2OhLbijlBg5RcAiqMTAwNjQ0IExJQ0VOU0UAwZK9aiTqGrAdeGhuQXyL3Hw9GX8xMDA2NDQgYmluYXJ5LmpwZwDVwPSrgRiXyt8DrsNYrmDSH5HFDTQwMDAwIGdvAKOXcadlH5f69ccuCCJNhX/DUTPbNDAwMDAganNvbgBah35qkGonQ61uRdmcF5NkKq+O2jQwMDAwIHBocABYavVn0Ltedx5JvdlDT14Pt20l+jQwMDAwIHZlbmRvcgDPSqOziXT7fYHzZ8CDD3141lq4aw=="

func (s *ObjectsSuite) TestParseTree(c *C) {
	data, _ := base64.StdEncoding.DecodeString(TreeFixture)
	tree, err := ParseTree(data)
	c.Assert(err, IsNil)

	c.Assert(tree.Entries, HasLen, 8)
	c.Assert(tree.Entries[0].Name, Equals, ".gitignore")
	c.Assert(tree.Entries[0].Hash.String(), Equals, "32858aad3c383ed1ff0a0f9bdf231d54a00c9e88")
}

func (s *ObjectsSuite) TestTreeHash(c *C) {
	data, _ := base64.StdEncoding.DecodeString(TreeFixture)
	tree, err := ParseTree(data)

	c.Assert(err, IsNil)
	c.Assert(tree.Hash().String(), Equals, "a8d315b2b1c615d43042c3a62402b8a54288cf5c")
}

func (s *ObjectsSuite) TestBlobHash(c *C) {
	blob, err := ParseBlob([]byte{'F', 'O', 'O'})
	c.Assert(err, IsNil)

	c.Assert(blob.Len, Equals, 3)
	c.Assert(blob.Hash().String(), Equals, "d96c7efbfec2814ae0301ad054dc8d9fc416c9b5")
}

func (s *ObjectsSuite) TestParseSignature(c *C) {
	cases := map[string]Signature{
		`Foo Bar <foo@bar.com> 1257894000 +0100`: {
			Name:  "Foo Bar",
			Email: "foo@bar.com",
			When:  time.Unix(1257894000, 0),
		},
		`Foo Bar <> 1257894000 +0100`: {
			Name:  "Foo Bar",
			Email: "",
			When:  time.Unix(1257894000, 0),
		},
		` <> 1257894000`: {
			Name:  "",
			Email: "",
			When:  time.Unix(1257894000, 0),
		},
		`Foo Bar <foo@bar.com>`: {
			Name:  "Foo Bar",
			Email: "foo@bar.com",
			When:  time.Time{},
		},
		``: {
			Name:  "",
			Email: "",
			When:  time.Time{},
		},
		`<`: {
			Name:  "",
			Email: "",
			When:  time.Time{},
		},
	}

	for raw, exp := range cases {
		got := ParseSignature([]byte(raw))
		c.Assert(got.Name, Equals, exp.Name)
		c.Assert(got.Email, Equals, exp.Email)
		c.Assert(got.When.Unix(), Equals, exp.When.Unix())
	}
}
