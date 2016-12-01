package revision

import (
	"bytes"

	. "gopkg.in/check.v1"
)

type ParserSuite struct{}

var _ = Suite(&ParserSuite{})

func (s *ParserSuite) TestScan(c *C) {
	parser := newParser(bytes.NewBufferString("Hello world !"))

	expected := []struct {
		t token
		s string
	}{
		{
			word,
			"Hello",
		},
		{
			space,
			" ",
		},
		{
			word,
			"world",
		},
		{
			space,
			" ",
		},
		{
			emark,
			"!",
		},
	}

	for i := 0; ; {
		tok, str := parser.scan()

		if tok == eof {
			return
		}

		c.Assert(str, Equals, expected[i].s)
		c.Assert(tok, Equals, expected[i].t)

		i++
	}
}

func (s *ParserSuite) TestUnscan(c *C) {
	parser := newParser(bytes.NewBufferString("Hello world !"))

	tok, str := parser.scan()

	c.Assert(str, Equals, "Hello")
	c.Assert(tok, Equals, word)

	parser.unscan()

	tok, str = parser.scan()

	c.Assert(str, Equals, "Hello")
	c.Assert(tok, Equals, word)
}

func (s *ParserSuite) TestParseAtSuffixWithValidExpression(c *C) {
	datas := map[string]atSuffixer{
		"{1}": atSuffixReflog{1},
	}

	for d, expected := range datas {
		parser := newParser(bytes.NewBufferString(d))

		result, err := parser.parseAtSuffix()

		c.Assert(err, Equals, nil)
		c.Assert(result, DeepEquals, expected)
	}
}

func (s *ParserSuite) TestParseAtSuffixWithUnValidExpression(c *C) {
	datas := map[string]error{
		"test}":  &ErrInvalidRevision{`"test" found must be "{" after @`},
		"{test}": &ErrInvalidRevision{`invalid expression "test" in @{} structure`},
	}

	for s, e := range datas {
		parser := newParser(bytes.NewBufferString(s))

		_, err := parser.parseAtSuffix()

		c.Assert(err, DeepEquals, e)
	}
}

func (s *ParserSuite) TestParseRevSuffixWithValidExpression(c *C) {
	datas := map[string][]revSuffixer{
		"^": []revSuffixer{revSuffixPath{"^", 1}},
		"~": []revSuffixer{revSuffixPath{"~", 1}},
		"~^^1~2~10~^100^~1000": []revSuffixer{
			revSuffixPath{"~", 1},
			revSuffixPath{"^", 1},
			revSuffixPath{"^", 1},
			revSuffixPath{"~", 2},
			revSuffixPath{"~", 10},
			revSuffixPath{"~", 1},
			revSuffixPath{"^", 100},
			revSuffixPath{"^", 1},
			revSuffixPath{"~", 1000},
		},
		"^{}": []revSuffixer{
			revSuffixType{"tag"},
		},
		"^{commit}": []revSuffixer{
			revSuffixType{"commit"},
		},
		"^{tree}": []revSuffixer{
			revSuffixType{"tree"},
		},
		"^{blob}": []revSuffixer{
			revSuffixType{"blob"},
		},
		"^{tag}": []revSuffixer{
			revSuffixType{"tag"},
		},
		"^{object}": []revSuffixer{
			revSuffixType{"object"},
		},
		"^{/hello world !}": []revSuffixer{
			revSuffixReg{"hello world !", false},
		},
		"^{/!-hello world !}": []revSuffixer{
			revSuffixReg{"hello world !", true},
		},
		"^{/!! hello world !}": []revSuffixer{
			revSuffixReg{"! hello world !", false},
		},
		"^5^~3^{/!! hello world !}~2^{/test}^": []revSuffixer{
			revSuffixPath{"^", 5},
			revSuffixPath{"^", 1},
			revSuffixPath{"~", 3},
			revSuffixReg{"! hello world !", false},
			revSuffixPath{"~", 2},
			revSuffixReg{"test", false},
			revSuffixPath{"^", 1},
		},
	}

	for d, expected := range datas {
		parser := newParser(bytes.NewBufferString(d))

		result, err := parser.parseRevSuffix()

		c.Assert(err, Equals, nil)
		c.Assert(result, DeepEquals, expected)
	}
}

func (s *ParserSuite) TestParseRevSuffixWithUnValidExpression(c *C) {
	datas := map[string]error{
		"a":         &ErrInvalidRevision{`"a" is not a valid revision suffix component`},
		"~^~10a^10": &ErrInvalidRevision{`"a" is not a valid revision suffix component`},
		"^{test}":   &ErrInvalidRevision{`"test" is not a valid revision suffix brace component`},
		"^{/!test}": &ErrInvalidRevision{`revision suffix brace component sequences starting with "/!" others than those defined are reserved`},
	}

	for s, e := range datas {
		parser := newParser(bytes.NewBufferString(s))

		_, err := parser.parseRevSuffix()

		c.Assert(err, DeepEquals, e)
	}
}

func (s *ParserSuite) TestParseRefWithValidName(c *C) {
	datas := []string{
		"lock",
		"master",
		"v1.0.0",
		"refs/stash",
		"refs/tags/v1.0.0",
		"refs/heads/master",
		"refs/remotes/test",
		"refs/remotes/origin/HEAD",
		"refs/remotes/origin/master",
	}

	for _, d := range datas {
		parser := newParser(bytes.NewBufferString(d))

		result, err := parser.parseRef()

		c.Assert(err, Equals, nil)
		c.Assert(result, Equals, ref(d))
	}
}

func (s *ParserSuite) TestParseRefWithUnvalidName(c *C) {
	datas := map[string]error{
		".master":                     &ErrInvalidRevision{`must not start with "."`},
		"/master":                     &ErrInvalidRevision{`must not start with "/"`},
		"master/":                     &ErrInvalidRevision{`must not end with "/"`},
		"master.":                     &ErrInvalidRevision{`must not end with "."`},
		"refs/remotes/.origin/HEAD":   &ErrInvalidRevision{`must not contains "/."`},
		"test..test":                  &ErrInvalidRevision{`must not contains ".."`},
		"test..":                      &ErrInvalidRevision{`must not contains ".."`},
		"test test":                   &ErrInvalidRevision{`must not contains " "`},
		"test*test":                   &ErrInvalidRevision{`must not contains "*"`},
		"test?test":                   &ErrInvalidRevision{`must not contains "?"`},
		"test\\test":                  &ErrInvalidRevision{`must not contains "\"`},
		"test[test":                   &ErrInvalidRevision{`must not contains "["`},
		"te//st":                      &ErrInvalidRevision{`must not contains consecutively "/"`},
		"refs/remotes/test.lock/HEAD": &ErrInvalidRevision{`cannot end with .lock`},
		"test.lock":                   &ErrInvalidRevision{`cannot end with .lock`},
	}

	for s, e := range datas {
		parser := newParser(bytes.NewBufferString(s))

		_, err := parser.parseRef()

		c.Assert(err, DeepEquals, e)
	}
}
