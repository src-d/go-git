package pktlines_test

import (
	"io/ioutil"
	"strings"
	"testing"

	"gopkg.in/src-d/go-git.v3/formats/packp/pktline"
	"gopkg.in/src-d/go-git.v4/formats/packp/pktlines"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuitePktLine struct {
}

var _ = Suite(&SuitePktLine{})

func (s *SuitePktLine) TestNewIsEmpty(c *C) {
	p := pktlines.New()

	b, err := ioutil.ReadAll(p.R)
	c.Assert(err, IsNil)
	c.Assert(b, DeepEquals, []byte{})
}

func (s *SuitePktLine) TestAddFlush(c *C) {
	p := pktlines.New()
	p.AddFlush()

	b, err := ioutil.ReadAll(p.R)
	c.Assert(err, IsNil)
	c.Assert(string(b), DeepEquals, "0000")
}

func (s *SuitePktLine) TestAdd(c *C) {
	for i, test := range [...]struct {
		input    [][]byte
		expected []byte
	}{
		{
			input: [][]byte{
				[]byte("hello\n"),
			},
			expected: []byte("000ahello\n"),
		}, {
			input: [][]byte{
				[]byte("hello\n"),
				[]byte("world!\n"),
				[]byte("foo"),
			},
			expected: []byte("000ahello\n000bworld!\n0007foo"),
		}, {
			input: [][]byte{
				[]byte(strings.Repeat("a", pktline.MaxPayloadSize)),
			},
			expected: []byte(
				"fff0" + strings.Repeat("a", pktline.MaxPayloadSize)),
		}, {
			input: [][]byte{
				[]byte(strings.Repeat("a", pktline.MaxPayloadSize)),
				[]byte(strings.Repeat("b", pktline.MaxPayloadSize)),
			},
			expected: []byte(
				"fff0" + strings.Repeat("a", pktline.MaxPayloadSize) +
					"fff0" + strings.Repeat("b", pktline.MaxPayloadSize)),
		},
	} {
		p := pktlines.New()
		err := p.Add(test.input...)
		c.Assert(err, IsNil, Commentf("input %d = %v", i, test.input))

		obtained, err := ioutil.ReadAll(p.R)
		c.Assert(err, IsNil, Commentf("input %d = %v", i, test.input))

		c.Assert(obtained, DeepEquals, test.expected,
			Commentf("input %d = %v", i, test.input))
	}
}

func (s *SuitePktLine) TestAddErrEmptyPayload(c *C) {
	for _, input := range [...][][]byte{
		[][]byte{
			[]byte{},
		},
		[][]byte{
			[]byte(nil),
		},
		[][]byte{
			[]byte("hello world!"),
			[]byte{},
		},
		[][]byte{
			[]byte{},
			[]byte("hello world!"),
		},
	} {
		p := pktlines.New()
		err := p.Add(input...)
		c.Assert(err, Equals, pktlines.ErrEmptyPayload)
	}
}

func (s *SuitePktLine) TestAddErrPayloadTooLong(c *C) {
	for _, input := range [...][][]byte{
		[][]byte{
			[]byte(strings.Repeat("a", pktline.MaxPayloadSize+1)),
		},
		[][]byte{
			[]byte("hello world!"),
			[]byte(strings.Repeat("a", pktline.MaxPayloadSize+1)),
		},
		[][]byte{
			[]byte("hello world!"),
			[]byte(strings.Repeat("a", pktline.MaxPayloadSize+1)),
			[]byte("foo"),
		},
	} {
		p := pktlines.New()
		err := p.Add(input...)
		c.Assert(err, Equals, pktlines.ErrPayloadTooLong,
			Commentf("%v\n", input))
	}
}

func (s *SuitePktLine) TestAddString(c *C) {
	for i, test := range [...]struct {
		input    []string
		expected []byte
	}{
		{
			input: []string{
				"hello\n",
			},
			expected: []byte("000ahello\n"),
		}, {
			input: []string{
				"hello\n",
				"world!\n",
				"foo",
			},
			expected: []byte("000ahello\n000bworld!\n0007foo"),
		}, {
			input: []string{
				strings.Repeat("a", pktline.MaxPayloadSize),
			},
			expected: []byte(
				"fff0" + strings.Repeat("a", pktline.MaxPayloadSize)),
		}, {
			input: []string{
				strings.Repeat("a", pktline.MaxPayloadSize),
				strings.Repeat("b", pktline.MaxPayloadSize),
			},
			expected: []byte(
				"fff0" + strings.Repeat("a", pktline.MaxPayloadSize) +
					"fff0" + strings.Repeat("b", pktline.MaxPayloadSize)),
		},
	} {
		p := pktlines.New()
		err := p.AddString(test.input...)
		c.Assert(err, IsNil, Commentf("input %d = %v", i, test.input))

		obtained, err := ioutil.ReadAll(p.R)
		c.Assert(err, IsNil, Commentf("input %d = %v", i, test.input))

		c.Assert(obtained, DeepEquals, test.expected,
			Commentf("input %d = %v", i, test.input))
	}
}

func (s *SuitePktLine) TestAddStringErrEmptyPayload(c *C) {
	for _, input := range [...][]string{
		[]string{""},
		[]string{"hello world!", ""},
		[]string{"", "hello world!"},
	} {
		p := pktlines.New()
		err := p.AddString(input...)
		c.Assert(err, Equals, pktlines.ErrEmptyPayload)
	}
}

func (s *SuitePktLine) TestAddStringErrPayloadTooLong(c *C) {
	for _, input := range [...][]string{
		[]string{
			strings.Repeat("a", pktline.MaxPayloadSize+1),
		},
		[]string{
			"hello world!",
			strings.Repeat("a", pktline.MaxPayloadSize+1),
		},
		[]string{
			"hello world!",
			strings.Repeat("a", pktline.MaxPayloadSize+1),
			"foo",
		},
	} {
		p := pktlines.New()
		err := p.AddString(input...)
		c.Assert(err, Equals, pktlines.ErrPayloadTooLong,
			Commentf("%v\n", input))
	}
}

/*
func (s *SuitePktLine) TestNewFromStrings(c *C) {
	for _, test := range [...]struct {
		input    []string
		expected []byte
	}{
		{
			input:    []string(nil),
			expected: []byte{},
		}, {
			input:    []string{},
			expected: []byte{},
		}, {
			input:    []string{""},
			expected: []byte("0000"),
		}, {
			input:    []string{"hello\n"},
			expected: []byte("000ahello\n"),
		}, {
			input:    []string{"hello\n", "world!\n", "", "foo", ""},
			expected: []byte("000ahello\n000bworld!\n00000007foo0000"),
		}, {
			input: []string{
				strings.Repeat("a", pktline.MaxPayloadSize),
			},
			expected: []byte("fff0" + strings.Repeat("a", pktline.MaxPayloadSize)),
		},
	} {
		r, err := pktline.NewFromStrings(test.input...)
		c.Assert(err, IsNil)

		obtained, err := ioutil.ReadAll(r)
		c.Assert(err, IsNil)

		c.Assert(obtained, DeepEquals, test.expected,
			Commentf("input = %v\n", test.input))
	}
}

func (s *SuitePktLine) TestNewFromStringsErrPayloadTooLong(c *C) {
	for _, input := range [...][]string{
		[]string{
			strings.Repeat("a", pktline.MaxPayloadSize+1),
		},
		[]string{
			"hello world!",
			"",
			strings.Repeat("a", pktline.MaxPayloadSize+1),
		},
		[]string{
			"hello world!",
			strings.Repeat("a", pktline.MaxPayloadSize+1),
			"foo",
		},
	} {
		_, err := pktline.NewFromStrings(input...)

		c.Assert(err, Equals, pktline.ErrPayloadTooLong,
			Commentf("%v\n", input))
	}
}

func ExampleNew() {
	// These are the payloads we want to turn into pkt-lines,
	// the empty slice at the end will generate a flush-pkt.
	payloads := [][]byte{
		[]byte{'h', 'e', 'l', 'l', 'o', '\n'},
		[]byte{'w', 'o', 'r', 'l', 'd', '!', '\n'},
		[]byte{},
	}

	// Create the pkt-lines, ignoring errors...
	pktlines, _ := pktline.New(payloads...)

	// Send the raw data to stdout, ignoring errors...
	_, _ = io.Copy(os.Stdout, pktlines)

	// Output: 000ahello
	// 000bworld!
	// 0000
}

func ExampleNewFromStrings() {
	// These are the payloads we want to turn into pkt-lines,
	// the empty string at the end will generate a flush-pkt.
	payloads := []string{
		"hello\n",
		"world!\n",
		"",
	}

	// Create the pkt-lines, ignoring errors...
	pktlines, _ := pktline.NewFromStrings(payloads...)

	// Send the raw data to stdout, ignoring errors...
	_, _ = io.Copy(os.Stdout, pktlines)

	// Output: 000ahello
	// 000bworld!
	// 0000
}
*/