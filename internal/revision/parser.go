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

// revisioner represents a revision component
type revisioner interface {
}

// ref represents a reference name
type ref string

// tildeSuffix represents ~, ~{n}
type tildeSuffixPath struct {
	deep int
}

// caretSuffixPath represents ^, ^{n}
type caretSuffixPath struct {
	deep int
}

// caretSuffixReg represents ^{/foo bar} suffix
type caretSuffixReg struct {
	re     string
	negate bool
}

// caretSuffixType represents ^{commit} suffix
type caretSuffixType struct {
	object string
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

// parse explode a revision string into components
func (p *parser) parse() ([]revisioner, error) {
	// var tok token
	var rev revisioner
	var revs []revisioner
	var err error

	for {
		tok, _ := p.scan()

		switch tok {
		case at:
			p.unscan()
			rev, err = p.parseAtSuffix()
		case tilde:
			p.unscan()
			rev, err = p.parseTildeSuffix()
		case caret:
			p.unscan()
			rev, err = p.parseCaretSuffix()
		case eof:
			return revs, nil
		default:
			p.unscan()
			rev, err = p.parseRef()
		}

		if err != nil {
			return []revisioner{}, nil
		}

		revs = append(revs, rev)
	}
}

// parseAtSuffix extract part following @
func (p *parser) parseAtSuffix() (revisioner, error) {
	var tok, nextTok token
	var lit, nextLit string

	tok, lit = p.scan()

	if tok != at {
		return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" found must be "@"`, lit)}
	}

	tok, lit = p.scan()

	if tok != obrace {
		return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" found must be "{" after @`, lit)}
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
			return []revisioner{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a number`, lit)}
		}

		return atSuffixReflog{n}, nil
	case tok == minus && nextTok == number:
		n, err := strconv.Atoi(nextLit)

		if err != nil {
			return []revisioner{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a number`, nextLit)}
		}

		t, _ := p.scan()

		if t != cbrace {
			return nil, &ErrInvalidRevision{fmt.Sprintf(`missing "}" in @{-n} structure`)}
		}

		return atSuffixCheckout{n}, nil
	}

	return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`invalid expression "%s" in @{} structure`, lit)}
}

// parseTildeSuffix extract part following tilde
func (p *parser) parseTildeSuffix() (revisioner, error) {
	var tok token
	var lit string

	tok, lit = p.scan()

	if tok != tilde {
		return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" found must be "~"`, lit)}
	}

	tok, lit = p.scan()

	switch {
	case tok == number:
		n, err := strconv.Atoi(lit)

		if err != nil {
			return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a number`, lit)}
		}

		return tildeSuffixPath{n}, nil
	case tok == tilde || tok == caret || tok == eof:
		p.unscan()
		return tildeSuffixPath{1}, nil
	default:
		return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a valid revision suffix component`, lit)}
	}
}

// parseCaretSuffix extract part following caret
func (p *parser) parseCaretSuffix() (revisioner, error) {
	var tok token
	var lit string

	tok, lit = p.scan()

	if tok != caret {
		return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" found must be "^"`, lit)}
	}

	tok, lit = p.scan()

	switch {
	case tok == obrace:
		p.unscan()

		r, err := p.parseCaretSuffixWithBraces()

		if err != nil {
			return (revisioner)(struct{}{}), err
		}

		return r, nil
	case tok == number:
		n, err := strconv.Atoi(lit)

		if err != nil {
			return []revisioner{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a number`, lit)}
		}

		return caretSuffixPath{n}, nil
	case tok == caret || tok == tilde || tok == eof:
		p.unscan()
		return caretSuffixPath{1}, nil
	default:
		return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a valid revision suffix component`, lit)}
	}
}

// parseCaretSuffixWithBraces extract suffix between braces following caret
// todo : add regexp checker
func (p *parser) parseCaretSuffixWithBraces() (revisioner, error) {
	var tok, nextTok token
	var lit, _ string
	start := true
	reg := caretSuffixReg{}

	tok, lit = p.scan()

	if tok != obrace {
		return []revisioner{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" found must be "{" after ^`, lit)}
	}

	for {
		tok, lit = p.scan()
		nextTok, _ = p.scan()

		switch {
		case tok == word && nextTok == cbrace && (lit == "commit" || lit == "tree" || lit == "blob" || lit == "tag" || lit == "object"):
			return caretSuffixType{lit}, nil
		case reg.re == "" && tok == cbrace:
			return caretSuffixType{"tag"}, nil
		case reg.re == "" && tok == emark && nextTok == emark:
			reg.re += lit
		case reg.re == "" && tok == emark && nextTok == minus:
			reg.negate = true
		case reg.re == "" && tok == emark:
			return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`revision suffix brace component sequences starting with "/!" others than those defined are reserved`)}
		case reg.re == "" && tok == slash:
			p.unscan()
		case tok != slash && start:
			return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a valid revision suffix brace component`, lit)}
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
func (p *parser) parseRef() (revisioner, error) {
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
			p.unscan()
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
