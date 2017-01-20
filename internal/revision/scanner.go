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

// Scan extracts tokens and their strings counterpart
// from the reader
func (s *scanner) scan() (token, string, error) {
	ch, _, err := s.r.ReadRune()

	if err != nil && err != io.EOF {
		return tokenError, "", err
	}

	switch ch {
	case zeroRune:
		return eof, "", nil
	case ':':
		return colon, string(ch), nil
	case '~':
		return tilde, string(ch), nil
	case '^':
		return caret, string(ch), nil
	case '.':
		return dot, string(ch), nil
	case '/':
		return slash, string(ch), nil
	case '{':
		return obrace, string(ch), nil
	case '}':
		return cbrace, string(ch), nil
	case '-':
		return minus, string(ch), nil
	case '@':
		return at, string(ch), nil
	case '\\':
		return aslash, string(ch), nil
	case '?':
		return qmark, string(ch), nil
	case '*':
		return asterisk, string(ch), nil
	case '[':
		return obracket, string(ch), nil
	case '!':
		return emark, string(ch), nil
	}

	if unicode.IsSpace(ch) {
		return space, string(ch), nil
	}

	if unicode.IsControl(ch) {
		return control, string(ch), nil
	}

	if unicode.IsLetter(ch) {
		var data []rune
		data = append(data, ch)

		for {
			c, _, err := s.r.ReadRune()

			if c == zeroRune {
				break
			}

			if err != nil {
				return tokenError, "", err
			}

			if unicode.IsLetter(c) {
				data = append(data, c)
			} else {
				err := s.r.UnreadRune()

				if err != nil {
					return tokenError, "", err
				}

				return word, string(data), nil
			}
		}

		return word, string(data), nil
	}

	if unicode.IsNumber(ch) {
		var data []rune
		data = append(data, ch)

		for {
			c, _, err := s.r.ReadRune()

			if c == zeroRune {
				break
			}

			if err != nil {
				return tokenError, "", err
			}

			if unicode.IsNumber(c) {
				data = append(data, c)
			} else {
				err := s.r.UnreadRune()

				if err != nil {
					return tokenError, "", err
				}

				return number, string(data), nil
			}
		}

		return number, string(data), nil
	}

	return tokenError, string(ch), nil
}
