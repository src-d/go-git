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
			char,
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
