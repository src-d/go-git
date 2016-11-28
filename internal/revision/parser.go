package revision

import (
	"fmt"
	"io"
)

// ErrInvalidRevision is emitted if string doesn't match valid revision
type ErrInvalidRevision struct {
	s string
}

func (e *ErrInvalidRevision) Error() string {
	return "Revision invalid : " + e.s
}

// parser represents a parser.
type parser struct {
	s   *scanner
	buf struct {
		tok token
		lit string
		n   int
	}
}

// newParser returns a new instance of parser.
func newParser(r io.Reader) *parser {
	return &parser{s: newScanner(r)}
}

// scan returns the next token from the underlying scanner.
// If a token has been unscanned then read that instead.
func (p *parser) scan() (tok token, lit string) {
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}

	tok, lit = p.s.scan()

	p.buf.tok, p.buf.lit = tok, lit
	return
}

// unscan pushes the previously read token back onto the buffer.
func (p *parser) unscan() { p.buf.n = 1 }

// parseRevSuffix extract part following revision
func (p *parser) parseRevSuffix() ([]string, error) {
	var tok token
	var nextTok token
	var lit string
	var nextLit string
	var components []string

	for {
		tok, lit = p.scan()
		nextTok, nextLit = p.scan()

		switch {
		case (tok == caret || tok == tilde) && nextTok == number:
			components = append(components, lit+nextLit)
		case (tok == caret || tok == tilde):
			components = append(components, lit)
			p.unscan()
		case tok == eof:
			return components, nil
		default:
			return []string{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a valid revision suffix component`, lit)}
		}
	}
}

// parseRef extract reference name
func (p *parser) parseRef() (string, error) {
	var tok token
	var prevTok token
	var lit string
	var buf string
	var endOfRef bool

	for {
		tok, lit = p.scan()

		switch tok {
		case eof, at, colon, tilde, caret:
			endOfRef = true
		}

		err := p.checkRefFormat(tok, lit, prevTok, buf, endOfRef)

		if err != nil {
			return "", err
		}

		if endOfRef {
			return buf, nil
		}

		buf += lit
		prevTok = tok
	}
}

// checkRefFormat ensure reference name follow rules defined here :
// https://git-scm.com/docs/git-check-ref-format
func (p *parser) checkRefFormat(token token, literal string, previousToken token, buffer string, endOfRef bool) error {
	switch token {
	case aslash, space, control, qmark, asterisk, obracket:
		return &ErrInvalidRevision{fmt.Sprintf(`must not contains "%s"`, literal)}
	}

	switch {
	case (token == dot || token == slash) && buffer == "":
		return &ErrInvalidRevision{fmt.Sprintf(`must not start with "%s"`, literal)}
	case previousToken == slash && endOfRef:
		return &ErrInvalidRevision{`must not end with "/"`}
	case previousToken == dot && endOfRef:
		return &ErrInvalidRevision{`must not end with "."`}
	case token == dot && previousToken == slash:
		return &ErrInvalidRevision{`must not contains "/."`}
	case previousToken == dot && token == dot:
		return &ErrInvalidRevision{`must not contains ".."`}
	case previousToken == slash && token == slash:
		return &ErrInvalidRevision{`must not contains consecutively "/"`}
	case (token == slash || endOfRef) && len(buffer) > 4 && buffer[len(buffer)-5:] == ".lock":
		return &ErrInvalidRevision{"cannot end with .lock"}
	}

	return nil
}
