package pktlines_test

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/src-d/go-git.v4/formats/packp/pktlines"

	. "gopkg.in/check.v1"
)

type SuiteScanner struct{}

var _ = Suite(&SuiteScanner{})

func (s *SuiteScanner) TestInvalid(c *C) {
	for _, test := range [...]string{
		"0001", "0002", "0003", "0004",
		"0001asdfsadf", "0004foo",
		"fff1", "fff2",
		"gorka",
		"0", "003",
		"   5a", "5   a", "5   \n",
		"-001", "-000",
	} {
		r := strings.NewReader(test)
		sc := pktlines.NewScanner(r)
		_ = sc.Scan()
		c.Assert(sc.Err(), ErrorMatches, pktlines.ErrInvalidPktLen.Error(),
			Commentf("data = %q", test))
	}
}

func (s *SuiteScanner) TestEmptyReader(c *C) {
	r := strings.NewReader("")
	sc := pktlines.NewScanner(r)
	hasPayload := sc.Scan()
	c.Assert(hasPayload, Equals, false)
	c.Assert(sc.Err(), Equals, nil)
}

func (s *SuiteScanner) TestFlush(c *C) {
	p := pktlines.New()
	p.AddFlush()
	sc := pktlines.NewScanner(p.R)

	c.Assert(sc.Scan(), Equals, true)
	payload := sc.Bytes()
	c.Assert(len(payload), Equals, 0)
}

func (s *SuiteScanner) TestPktLineTooShort(c *C) {
	r := strings.NewReader("010cfoobar")

	sc := pktlines.NewScanner(r)

	c.Assert(sc.Scan(), Equals, false)
	c.Assert(sc.Err(), ErrorMatches, "unexpected EOF")
}

func (s *SuiteScanner) TestScanAndPayload(c *C) {
	for _, test := range [...]string{
		"a",
		"a\n",
		strings.Repeat("a", 100),
		strings.Repeat("a", 100) + "\n",
		strings.Repeat("\x00", 100),
		strings.Repeat("\x00", 100) + "\n",
		strings.Repeat("a", pktlines.MaxPayloadSize),
		strings.Repeat("a", pktlines.MaxPayloadSize-1) + "\n",
	} {
		p := pktlines.New()
		err := p.AddString(test)
		c.Assert(err, IsNil,
			Commentf("input len=%x, contents=%.10q\n", len(test), test))
		sc := pktlines.NewScanner(p.R)

		c.Assert(sc.Scan(), Equals, true,
			Commentf("test = %.20q...", test))
		obtained := sc.Bytes()
		c.Assert(obtained, DeepEquals, []byte(test),
			Commentf("in = %.20q out = %.20q", test, string(obtained)))
	}
}

func (s *SuiteScanner) TestSkip(c *C) {
	for _, test := range [...]struct {
		input    []string
		n        int
		expected []byte
	}{
		{
			input: []string{
				"first",
				"second",
				"third"},
			n:        1,
			expected: []byte("second"),
		},
		{
			input: []string{
				"first",
				"second",
				"third"},
			n:        2,
			expected: []byte("third"),
		},
	} {
		p := pktlines.New()
		err := p.AddString(test.input...)
		c.Assert(err, IsNil)
		sc := pktlines.NewScanner(p.R)
		for i := 0; i < test.n; i++ {
			c.Assert(sc.Scan(), Equals, true,
				Commentf("scan error = %s", sc.Err()))
		}
		c.Assert(sc.Scan(), Equals, true,
			Commentf("scan error = %s", sc.Err()))
		obtained := sc.Bytes()
		c.Assert(obtained, DeepEquals, test.expected,
			Commentf("\nin = %.20q\nout = %.20q\nexp = %.20q",
				test.input, obtained, test.expected))
	}
}

func (s *SuiteScanner) TestEOF(c *C) {
	p := pktlines.New()
	err := p.AddString("first", "second")
	c.Assert(err, IsNil)
	sc := pktlines.NewScanner(p.R)
	for sc.Scan() {
	}
	c.Assert(sc.Err(), IsNil)
}

// A section are several non flush-pkt lines followed by a flush-pkt, which
// how the git protocol sends long messages.
func (s *SuiteScanner) TestReadSomeSections(c *C) {
	nSections := 2
	nLines := 4
	data := sectionsExample(c, nSections, nLines)
	sc := pktlines.NewScanner(data)

	sectionCounter := 0
	lineCounter := 0
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			sectionCounter++
		}
		lineCounter++
	}
	c.Assert(sc.Err(), IsNil)
	c.Assert(sectionCounter, Equals, nSections)
	c.Assert(lineCounter, Equals, (1+nLines)*nSections)
}

// returns nSection sections, each of them with nLines pkt-lines (not
// counting the flush-pkt:
//
// 0009 0.0\n
// 0009 0.1\n
// ...
// 0000
// and so on
func sectionsExample(c *C, nSections, nLines int) io.Reader {
	p := pktlines.New()
	for section := 0; section < nSections; section++ {
		ss := []string{}
		for line := 0; line < nLines; line++ {
			line := fmt.Sprintf(" %d.%d\n", section, line)
			ss = append(ss, line)
		}
		err := p.AddString(ss...)
		c.Assert(err, IsNil)
		p.AddFlush()
	}

	return p.R
}

func ExampleScanner() {
	// A reader is needed as input.
	input := strings.NewReader("000ahello\n" +
		"000bworld!\n" +
		"0000",
	)

	// Create the scanner...
	s := pktlines.NewScanner(input)

	// and scan every pkt-line found in the input.
	for s.Scan() {
		payload := s.Bytes()
		if len(payload) == 0 { // zero sized payloads correspond to flush-pkts.
			fmt.Println("FLUSH-PKT DETECTED\n")
		} else { // otherwise, you will be able to access the full payload.
			fmt.Printf("PAYLOAD = %q\n", string(payload))
		}
	}

	// this will catch any error when reading from the input, if any.
	if s.Err() != nil {
		fmt.Println(s.Err())
	}

	// Output:
	// PAYLOAD = "hello\n"
	// PAYLOAD = "world!\n"
	// FLUSH-PKT DETECTED
}
