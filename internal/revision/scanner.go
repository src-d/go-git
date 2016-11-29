package revision

import (
	"bufio"
	"io"
	"unicode"
)

var zeroRune = rune(0)

// scanner represents a lexical scanner.
type scanner struct {
	r *bufio.Reader
}

// newScanner returns a new instance of scanner.
func newScanner(r io.Reader) *scanner {
	return &scanner{r: bufio.NewReader(r)}
}

// read reads the next rune from the bufferred reader.
// Returns the rune(0) if an error occurs (or io.EOF is returned).
func (s *scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return zeroRune
	}
	return ch
}

// unread places the previously read rune back on the reader.
func (s *scanner) unread() { _ = s.r.UnreadRune() }

// Scan extract tokens from an input
func (s *scanner) scan() (token, string) {
	ch := s.read()

	switch ch {
	case zeroRune:
		return eof, ""
	case ':':
		return colon, string(ch)
	case '~':
		return tilde, string(ch)
	case '^':
		return caret, string(ch)
	case '.':
		return dot, string(ch)
	case '/':
		return slash, string(ch)
	case '{':
		return obrace, string(ch)
	case '}':
		return cbrace, string(ch)
	case '-':
		return minus, string(ch)
	case '@':
		return at, string(ch)
	case '\\':
		return aslash, string(ch)
	case '?':
		return qmark, string(ch)
	case '*':
		return asterisk, string(ch)
	case '[':
		return obracket, string(ch)
	case '!':
		return emark, string(ch)
	}

	if unicode.IsSpace(ch) {
		return space, string(ch)
	}

	if unicode.IsControl(ch) {
		return control, string(ch)
	}

	if unicode.IsLetter(ch) {
		s.unread()

		return s.scanWord()
	}

	if unicode.IsNumber(ch) {
		s.unread()

		return s.scanNumber()
	}

	return char, string(ch)
}

// scanNumber return number token
func (s *scanner) scanNumber() (token, string) {
	var data string

	for c := s.read(); c != zeroRune; c = s.read() {
		if unicode.IsNumber(c) {
			data += string(c)
		} else {
			s.unread()
			return number, data
		}
	}

	return number, data
}

// scanWord return a word token
func (s *scanner) scanWord() (token, string) {
	var data string

	for c := s.read(); c != zeroRune; c = s.read() {
		if unicode.IsLetter(c) {
			data += string(c)
		} else {
			s.unread()
			return word, data
		}
	}

	return word, data
}
