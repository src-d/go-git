package revision

import (
	"fmt"
	"io"
	"strconv"
)

// ErrInvalidRevision is emitted if string doesn't match valid revision
type ErrInvalidRevision struct {
	s string
}

func (e *ErrInvalidRevision) Error() string {
	return "Revision invalid : " + e.s
}

// ref represents a reference name
type ref string

// revSuffixer represents a generic revision suffix
type revSuffixer interface {
}

// revSuffixPath represents ^ or ~ revision suffix
type revSuffixPath struct {
	suffix string
	deep   int
}

// revSuffixReg represents ^{/foo bar} revision suffix
type revSuffixReg struct {
	re     string
	negate bool
}

// revSuffixType represents ^{commit} revision suffix
type revSuffixType struct {
	object string
}

// atSuffixer represents generic suffix added to @
type atSuffixer interface {
}

// atSuffixReflog represents @{n}
type atSuffixReflog struct {
	deep int
}

// atSuffixCheckout represents @{-n}
type atSuffixCheckout struct {
	deep int
}

// atUpstream represents @{upstream}, @{u
type atUpstream struct {
	branchName string
}

// atPush represents @{push}
type atPush struct {
	branchName string
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

// parseAtSuffix extract part following @
func (p *parser) parseAtSuffix() (atSuffixer, error) {
	var tok, nextTok token
	var lit, nextLit string

	for {
		tok, lit = p.scan()

		if tok != obrace {
			return (atSuffixer)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" found must be "{" after @`, lit)}
		}

		tok, lit = p.scan()
		nextTok, nextLit = p.scan()

		switch {
		case tok == word && (lit == "u" || lit == "upstream") && nextTok == cbrace:
			return atUpstream{}, nil
		case tok == word && lit == "push" && nextTok == cbrace:
			return atPush{}, nil
		case tok == number && nextTok == cbrace:
			n, err := strconv.Atoi(lit)

			if err != nil {
				return []atSuffixer{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a number`, lit)}
			}

			return atSuffixReflog{n}, nil
		case tok == minus && nextTok == number:
			n, err := strconv.Atoi(nextLit)

			if err != nil {
				return []atSuffixer{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a number`, nextLit)}
			}

			t, _ := p.scan()

			if t != cbrace {
				return nil, &ErrInvalidRevision{fmt.Sprintf(`missing "}" in @{-n} structure`)}
			}

			return atSuffixCheckout{n}, nil
		}

		return (atSuffixer)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`invalid expression "%s" in @{} structure`, lit)}
	}
}

// parseRevSuffix extract part following revision
func (p *parser) parseRevSuffix() ([]revSuffixer, error) {
	var tok, nextTok token
	var lit, nextLit string
	var components []revSuffixer

	for {
		tok, lit = p.scan()
		nextTok, nextLit = p.scan()

		switch {
		case tok == caret && nextTok == obrace:
			r, err := p.parseRevSuffixInBraces()

			if err != nil {
				return []revSuffixer{}, err
			}

			components = append(components, r)
		case (tok == caret || tok == tilde) && nextTok == number:
			n, err := strconv.Atoi(nextLit)

			if err != nil {
				return []revSuffixer{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a number`, nextLit)}
			}

			r := revSuffixPath{lit, n}

			components = append(components, r)
		case (tok == caret || tok == tilde):
			components = append(components, revSuffixPath{lit, 1})
			p.unscan()
		case tok == eof:
			return components, nil
		default:
			return []revSuffixer{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a valid revision suffix component`, lit)}
		}
	}
}

// parseRevSuffixInBraces extract revision suffix between braces
// todo : add regexp checker
func (p *parser) parseRevSuffixInBraces() (revSuffixer, error) {
	var tok, nextTok token
	var lit, _ string
	start := true
	reg := revSuffixReg{}

	for {
		tok, lit = p.scan()
		nextTok, _ = p.scan()

		switch {
		case tok == word && nextTok == cbrace && (lit == "commit" || lit == "tree" || lit == "blob" || lit == "tag" || lit == "object"):
			return revSuffixType{lit}, nil
		case reg.re == "" && tok == cbrace:
			return revSuffixType{"tag"}, nil
		case reg.re == "" && tok == emark && nextTok == emark:
			reg.re += lit
		case reg.re == "" && tok == emark && nextTok == minus:
			reg.negate = true
		case reg.re == "" && tok == emark:
			return (revSuffixer)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`revision suffix brace component sequences starting with "/!" others than those defined are reserved`)}
		case reg.re == "" && tok == slash:
			p.unscan()
		case tok != slash && start:
			return (revSuffixer)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a valid revision suffix brace component`, lit)}
		case tok != cbrace:
			p.unscan()
			reg.re += lit
		case tok == cbrace:
			p.unscan()
			return reg, nil
		}

		start = false
	}
}

// parseRef extract reference name
func (p *parser) parseRef() (ref, error) {
	var tok, prevTok token
	var lit, buf string
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
			return ref(buf), nil
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
