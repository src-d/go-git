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

func (s *ParserSuite) TestParse(c *C) {
	datas := map[string]revisioner{
		"@": []revisioner{ref("HEAD")},
		"@~3": []revisioner{
			ref("HEAD"),
			tildePath{3},
		},
		"@{1}":  []revisioner{atReflog{1}},
		"@{-1}": []revisioner{atCheckout{1}},
		"master@{upstream}": []revisioner{
			ref("master"),
			atUpstream{},
		},
		"master@{push}": []revisioner{
			ref("master"),
			atPush{},
		},
		"HEAD^": []revisioner{
			ref("HEAD"),
			caretPath{1},
		},
		"master~3": []revisioner{
			ref("master"),
			tildePath{3},
		},
		"v0.99.8^{commit}": []revisioner{
			ref("v0.99.8"),
			caretType{"commit"},
		},
		"v0.99.8^{}": []revisioner{
			ref("v0.99.8"),
			caretType{"tag"},
		},
		"HEAD^{/fix nasty bug}": []revisioner{
			ref("HEAD"),
			caretReg{"fix nasty bug", false},
		},
		"master~1^{/update}~5~^^1": []revisioner{
			ref("master"),
			tildePath{1},
			caretReg{"update", false},
			tildePath{5},
			tildePath{1},
			caretPath{1},
			caretPath{1},
		},
	}

	for d, expected := range datas {
		parser := newParser(bytes.NewBufferString(d))

		result, err := parser.parse()

		c.Assert(err, Equals, nil)
		c.Assert(result, DeepEquals, expected)
	}
}

func (s *ParserSuite) TestParseAtWithValidExpression(c *C) {
	datas := map[string]revisioner{
		"@":           ref("HEAD"),
		"@{1}":        atReflog{1},
		"@{-1}":       atCheckout{1},
		"@{push}":     atPush{},
		"@{upstream}": atUpstream{},
		"@{u}":        atUpstream{},
	}

	for d, expected := range datas {
		parser := newParser(bytes.NewBufferString(d))

		result, err := parser.parseAt()

		c.Assert(err, Equals, nil)
		c.Assert(result, DeepEquals, expected)
	}
}

func (s *ParserSuite) TestParseAtWithUnValidExpression(c *C) {
	datas := map[string]error{
		"a":       &ErrInvalidRevision{`"a" found must be "@"`},
		"@{test}": &ErrInvalidRevision{`invalid expression "test" in @{} structure`},
		"@{-1":    &ErrInvalidRevision{`missing "}" in @{-n} structure`},
	}

	for s, e := range datas {
		parser := newParser(bytes.NewBufferString(s))

		_, err := parser.parseAt()

		c.Assert(err, DeepEquals, e)
	}
}

func (s *ParserSuite) TestParseCaretWithValidExpression(c *C) {
	datas := map[string]revisioner{
		"^":                    caretPath{1},
		"^3":                   caretPath{3},
		"^{}":                  caretType{"tag"},
		"^{commit}":            caretType{"commit"},
		"^{tree}":              caretType{"tree"},
		"^{blob}":              caretType{"blob"},
		"^{tag}":               caretType{"tag"},
		"^{object}":            caretType{"object"},
		"^{/hello world !}":    caretReg{"hello world !", false},
		"^{/!-hello world !}":  caretReg{"hello world !", true},
		"^{/!! hello world !}": caretReg{"! hello world !", false},
	}

	for d, expected := range datas {
		parser := newParser(bytes.NewBufferString(d))

		result, err := parser.parseCaret()

		c.Assert(err, Equals, nil)
		c.Assert(result, DeepEquals, expected)
	}
}

func (s *ParserSuite) TestParseCaretWithUnValidExpression(c *C) {
	datas := map[string]error{
		"a":         &ErrInvalidRevision{`"a" found must be "^"`},
		"^a":        &ErrInvalidRevision{`"a" is not a valid revision suffix component`},
		"^{test}":   &ErrInvalidRevision{`"test" is not a valid revision suffix brace component`},
		"^{/!test}": &ErrInvalidRevision{`revision suffix brace component sequences starting with "/!" others than those defined are reserved`},
	}

	for s, e := range datas {
		parser := newParser(bytes.NewBufferString(s))

		_, err := parser.parseCaret()

		c.Assert(err, DeepEquals, e)
	}
}

func (s *ParserSuite) TestParseTildeWithValidExpression(c *C) {
	datas := map[string]revisioner{
		"~3": tildePath{3},
		"~1": tildePath{1},
		"~":  tildePath{1},
	}

	for d, expected := range datas {
		parser := newParser(bytes.NewBufferString(d))

		result, err := parser.parseTilde()

		c.Assert(err, Equals, nil)
		c.Assert(result, DeepEquals, expected)
	}
}

func (s *ParserSuite) TestParseTildeWithUnValidExpression(c *C) {
	datas := map[string]error{
		"a":  &ErrInvalidRevision{`"a" found must be "~"`},
		"~a": &ErrInvalidRevision{`"a" is not a valid revision suffix component`},
	}

	for s, e := range datas {
		parser := newParser(bytes.NewBufferString(s))

		_, err := parser.parseTilde()

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
