package advrefs_test

import (
	"bytes"
	"strings"

	"gopkg.in/src-d/go-git.v4/core"
	"gopkg.in/src-d/go-git.v4/formats/packp"
	"gopkg.in/src-d/go-git.v4/formats/packp/advrefs"
	"gopkg.in/src-d/go-git.v4/formats/packp/pktline"

	. "gopkg.in/check.v1"
)

type SuiteAdvRefsDecode struct{}

var _ = Suite(&SuiteAdvRefsDecode{})

func (s *SuiteAdvRefsDecode) TestEmpty(c *C) {
	ar := advrefs.New()
	var buf bytes.Buffer
	d := advrefs.NewDecoder(&buf)

	err := d.Decode(ar)
	c.Assert(err, Equals, advrefs.ErrEmpty)
}

func (s *SuiteAdvRefs) TestShortForHash(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*too short")
}

func (s *SuiteAdvRefs) TestInvalidFirstHash(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796alberto2219af86ec6584e5 HEAD\x00multi_ack thin-pack\n",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*invalid hash.*")
}

func (s *SuiteAdvRefs) TestZeroId(c *C) {
	input := pktline.New()
	err := input.AddString(
		"0000000000000000000000000000000000000000 capabilities^{}\x00multi_ack thin-pack\n",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, IsNil)
	c.Assert(ar.Head, IsNil)
}

func (s *SuiteAdvRefs) TestParseMalformedZeroId(c *C) {
	input := pktline.New()
	err := input.AddString(
		"0000000000000000000000000000000000000000 wrong\x00multi_ack thin-pack\n",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*malformed zero-id.*")
}

func (s *SuiteAdvRefs) TestParseShortZeroId(c *C) {
	input := pktline.New()
	err := input.AddString(
		"0000000000000000000000000000000000000000 capabi",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*too short zero-id.*")
}

func (s *SuiteAdvRefs) TestParseHead(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, IsNil)
	c.Assert(*ar.Head, Equals,
		core.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
}

func (s *SuiteAdvRefs) TestParseFirstIsNotHead(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 refs/heads/master\x00",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, IsNil)
	c.Assert(ar.Head, IsNil)
	c.Assert(ar.Refs["refs/heads/master"], Equals,
		core.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5"))
}

func (s *SuiteAdvRefs) TestParseShortRef(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 H",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*too short.*")
}

func (s *SuiteAdvRefs) TestParseNoNULL(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEADofs-delta multi_ack",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*NULL not found.*")
}

func (s *SuiteAdvRefs) TestParseNoSpaceAfterHash(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5-HEAD\x00",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*no space after hash.*")
}

func (s *SuiteAdvRefs) TestParseNoCaps(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, IsNil)
	c.Assert(ar.Caps.IsEmpty(), Equals, true)
}

func (s *SuiteAdvRefs) TestParseCaps(c *C) {
	for _, test := range [...]struct {
		input []string
		caps  []packp.Capability
	}{
		{
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00",
				pktline.FlushString,
			},
			caps: []packp.Capability{},
		},
		{
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00\n",
				pktline.FlushString,
			},
			caps: []packp.Capability{},
		},
		{
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta",
				pktline.FlushString,
			},
			caps: []packp.Capability{
				{
					Name:   "ofs-delta",
					Values: []string(nil),
				},
			},
		},
		{
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta multi_ack",
				pktline.FlushString,
			},
			caps: []packp.Capability{
				{Name: "ofs-delta", Values: []string(nil)},
				{Name: "multi_ack", Values: []string(nil)},
			},
		},
		{
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta multi_ack\n",
				pktline.FlushString,
			},
			caps: []packp.Capability{
				{Name: "ofs-delta", Values: []string(nil)},
				{Name: "multi_ack", Values: []string(nil)},
			},
		},
		{
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00symref=HEAD:refs/heads/master agent=foo=bar\n",
				pktline.FlushString,
			},
			caps: []packp.Capability{
				{Name: "symref", Values: []string{"HEAD:refs/heads/master"}},
				{Name: "agent", Values: []string{"foo=bar"}},
			},
		},
		{
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00symref=HEAD:refs/heads/master agent=foo=bar agent=new-agent\n",
				pktline.FlushString,
			},
			caps: []packp.Capability{
				{Name: "symref", Values: []string{"HEAD:refs/heads/master"}},
				{Name: "agent", Values: []string{"foo=bar", "new-agent"}},
			},
		},
	} {
		input := pktline.New()
		err := input.AddString(test.input...)
		c.Assert(err, IsNil, Commentf("input = %q", test.input))

		ar := advrefs.New()
		d := advrefs.NewDecoder(input)

		err = d.Decode(ar)
		c.Assert(err, IsNil, Commentf("input = %q", test.input))

		for _, fixCap := range test.caps {
			c.Assert(ar.Caps.Supports(fixCap.Name), Equals, true,
				Commentf("input = %q, cap = %q", test.input, fixCap.Name))
			c.Assert(ar.Caps.Get(fixCap.Name).Values, DeepEquals, fixCap.Values,
				Commentf("input = %q, cap = %q", test.input, fixCap.Name))
		}
	}
}

func (s *SuiteAdvRefs) TestParseWithPrefix(c *C) {
	payloads := []string{
		"# this is a prefix\n",
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00foo\n",
		pktline.FlushString,
	}

	input := pktline.New()
	err := input.AddString(payloads...)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, IsNil)

	c.Assert(len(ar.Prefix), Equals, 1)
	c.Assert(ar.Prefix[0], DeepEquals, []byte("# this is a prefix"))
}

func (s *SuiteAdvRefs) TestParseWithPrefixAndFlush(c *C) {
	payloads := []string{
		"# this is a prefix\n",
		pktline.FlushString,
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00foo\n",
		pktline.FlushString,
	}

	input := pktline.New()
	err := input.AddString(payloads...)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, IsNil)

	c.Assert(len(ar.Prefix), Equals, 2)
	c.Assert(ar.Prefix[0], DeepEquals, []byte("# this is a prefix"))
	c.Assert(ar.Prefix[1], DeepEquals, []byte(pktline.FlushString))
}

func (s *SuiteAdvRefs) TestParseOtherRefs(c *C) {
	for _, test := range [...]struct {
		input  []string
		refs   map[string]core.Hash
		peeled map[string]core.Hash
	}{
		{
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
				pktline.FlushString,
			},
			refs:   map[string]core.Hash{},
			peeled: map[string]core.Hash{},
		}, {
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
				"1111111111111111111111111111111111111111 ref/foo",
				pktline.FlushString,
			},
			refs: map[string]core.Hash{
				"ref/foo": core.NewHash("1111111111111111111111111111111111111111"),
			},
			peeled: map[string]core.Hash{},
		}, {
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
				"1111111111111111111111111111111111111111 ref/foo\n",
				pktline.FlushString,
			},
			refs: map[string]core.Hash{
				"ref/foo": core.NewHash("1111111111111111111111111111111111111111"),
			},
			peeled: map[string]core.Hash{},
		}, {
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
				"1111111111111111111111111111111111111111 ref/foo\n",
				"2222222222222222222222222222222222222222 ref/bar",
				pktline.FlushString,
			},
			refs: map[string]core.Hash{
				"ref/foo": core.NewHash("1111111111111111111111111111111111111111"),
				"ref/bar": core.NewHash("2222222222222222222222222222222222222222"),
			},
			peeled: map[string]core.Hash{},
		}, {
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
				"1111111111111111111111111111111111111111 ref/foo^{}\n",
				pktline.FlushString,
			},
			refs: map[string]core.Hash{},
			peeled: map[string]core.Hash{
				"ref/foo": core.NewHash("1111111111111111111111111111111111111111"),
			},
		}, {
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
				"1111111111111111111111111111111111111111 ref/foo\n",
				"2222222222222222222222222222222222222222 ref/bar^{}",
				pktline.FlushString,
			},
			refs: map[string]core.Hash{
				"ref/foo": core.NewHash("1111111111111111111111111111111111111111"),
			},
			peeled: map[string]core.Hash{
				"ref/bar": core.NewHash("2222222222222222222222222222222222222222"),
			},
		}, {
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
				"a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n",
				"51b8b4fb32271d39fbdd760397406177b2b0fd36 refs/pull/10/head\n",
				"02b5a6031ba7a8cbfde5d65ff9e13ecdbc4a92ca refs/pull/100/head\n",
				"c284c212704c43659bf5913656b8b28e32da1621 refs/pull/100/merge\n",
				"3d6537dce68c8b7874333a1720958bd8db3ae8ca refs/pull/101/merge\n",
				"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11\n",
				"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11^{}\n",
				"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree\n",
				"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}\n",
				pktline.FlushString,
			},
			refs: map[string]core.Hash{
				"refs/heads/master":      core.NewHash("a6930aaee06755d1bdcfd943fbf614e4d92bb0c7"),
				"refs/pull/10/head":      core.NewHash("51b8b4fb32271d39fbdd760397406177b2b0fd36"),
				"refs/pull/100/head":     core.NewHash("02b5a6031ba7a8cbfde5d65ff9e13ecdbc4a92ca"),
				"refs/pull/100/merge":    core.NewHash("c284c212704c43659bf5913656b8b28e32da1621"),
				"refs/pull/101/merge":    core.NewHash("3d6537dce68c8b7874333a1720958bd8db3ae8ca"),
				"refs/tags/v2.6.11":      core.NewHash("5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c"),
				"refs/tags/v2.6.11-tree": core.NewHash("5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c"),
			},
			peeled: map[string]core.Hash{
				"refs/tags/v2.6.11":      core.NewHash("c39ae07f393806ccf406ef966e9a15afc43cc36a"),
				"refs/tags/v2.6.11-tree": core.NewHash("c39ae07f393806ccf406ef966e9a15afc43cc36a"),
			},
		},
	} {
		input := pktline.New()
		err := input.AddString(test.input...)
		c.Assert(err, IsNil)

		comment := Commentf("input = %q", test.input)

		ar := advrefs.New()
		d := advrefs.NewDecoder(input)

		err = d.Decode(ar)
		c.Assert(err, IsNil, comment)

		c.Assert(ar.Refs, DeepEquals, test.refs, comment)
		c.Assert(ar.Peeled, DeepEquals, test.peeled, comment)
	}
}

func (s *SuiteAdvRefs) TestParseMalformedOtherRefsNoSpace(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00multi_ack thin-pack\n",
		"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8crefs/tags/v2.6.11\n",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*malformed ref data.*")
}

func (s *SuiteAdvRefs) TestParseMalformedOtherRefsMultipleSpaces(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00multi_ack thin-pack\n",
		"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags v2.6.11\n",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*malformed ref data.*")
}

func (s *SuiteAdvRefs) TestParseShallow(c *C) {
	for _, test := range [...]struct {
		input    []string
		shallows []core.Hash
	}{
		{
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
				"a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n",
				"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree\n",
				"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}\n",
				pktline.FlushString,
			},
			shallows: []core.Hash{},
		}, {
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
				"a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n",
				"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree\n",
				"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}\n",
				"shallow 1111111111111111111111111111111111111111\n",
				pktline.FlushString,
			},
			shallows: []core.Hash{core.NewHash("1111111111111111111111111111111111111111")},
		}, {
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
				"a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n",
				"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree\n",
				"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}\n",
				"shallow 1111111111111111111111111111111111111111\n",
				"shallow 2222222222222222222222222222222222222222\n",
				pktline.FlushString,
			},
			shallows: []core.Hash{
				core.NewHash("1111111111111111111111111111111111111111"),
				core.NewHash("2222222222222222222222222222222222222222"),
			},
		},
	} {
		input := pktline.New()
		err := input.AddString(test.input...)
		c.Assert(err, IsNil)

		comment := Commentf("input = %q", test.input)

		ar := advrefs.New()
		d := advrefs.NewDecoder(input)

		err = d.Decode(ar)
		c.Assert(err, IsNil, comment)

		c.Assert(ar.Shallows, DeepEquals, test.shallows, comment)
	}
}

func (s *SuiteAdvRefs) TestParseInvalidShallowHash(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
		"a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n",
		"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree\n",
		"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}\n",
		"shallow 11111111alcortes111111111111111111111111\n",
		"shallow 2222222222222222222222222222222222222222\n",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*invalid hash text.*")
}

func (s *SuiteAdvRefs) TestParseGarbageAfterShallow(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
		"a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n",
		"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree\n",
		"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}\n",
		"shallow 1111111111111111111111111111111111111111\n",
		"shallow 2222222222222222222222222222222222222222\n",
		"b5be40b90dbaa6bd337f3b77de361bfc0723468b refs/tags/v4.4",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*malformed shallow prefix.*")
}

func (s *SuiteAdvRefs) TestParseMalformedShallowHash(c *C) {
	input := pktline.New()
	err := input.AddString(
		"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n",
		"a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n",
		"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree\n",
		"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}\n",
		"shallow 1111111111111111111111111111111111111111\n",
		"shallow 2222222222222222222222222222222222222222 malformed\n",
		pktline.FlushString,
	)
	c.Assert(err, IsNil)

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err = d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*malformed shallow hash.*")
}

func (s *SuiteAdvRefs) TestParseEOFRefs(c *C) {
	input := strings.NewReader("" +
		"005b6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n" +
		"003fa6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n" +
		"00355dc01c595e6c6ec9ccda4f6ffbf614e4d92bb0c7 refs/foo\n")

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err := d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*invalid pkt-len.*")
}

func (s *SuiteAdvRefs) TestParseEOFShallows(c *C) {
	input := strings.NewReader("" +
		"005b6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00ofs-delta symref=HEAD:/refs/heads/master\n" +
		"003fa6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n" +
		"00445dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree\n" +
		"0047c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}\n" +
		"0035shallow 1111111111111111111111111111111111111111\n" +
		"0034shallow 222222222222222222222222")

	ar := advrefs.New()
	d := advrefs.NewDecoder(input)

	err := d.Decode(ar)
	c.Assert(err, ErrorMatches, ".*unexpected EOF.*")
}
