package revision

import (
	"fmt"
	"io"
	"strconv"
	"time"
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

// tildePath represents ~, ~{n}
type tildePath struct {
	deep int
}

// caretPath represents ^, ^{n}
type caretPath struct {
	deep int
}

// caretReg represents ^{/foo bar}
type caretReg struct {
	re     string
	negate bool
}

// caretType represents ^{commit}
type caretType struct {
	object string
}

// atReflog represents @{n}
type atReflog struct {
	deep int
}

// atCheckout represents @{-n}
type atCheckout struct {
	deep int
}

// atUpstream represents @{upstream}, @{u}
type atUpstream struct {
	branchName string
}

// atPush represents @{push}
type atPush struct {
	branchName string
}

// atDate represents @{"2006-01-02T15:04:05Z"}
type atDate struct {
	date time.Time
}

// colonReg represents :/foo bar
type colonReg struct {
	re     string
	negate bool
}

// colonPath represents :../<path> :./<path> :<path>
type colonPath struct {
	path  string
	stage int
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
	var rev revisioner
	var revs []revisioner
	var err error

	for {
		tok, _ := p.scan()

		switch tok {
		case at:
			p.unscan()
			rev, err = p.parseAt()
		case tilde:
			p.unscan()
			rev, err = p.parseTilde()
		case caret:
			p.unscan()
			rev, err = p.parseCaret()
		case colon:
			p.unscan()
			rev, err = p.parseColon()
		case eof:
			err = p.validateFullRevision(&revs)

			if err != nil {
				return []revisioner{}, err
			}

			return revs, nil
		default:
			p.unscan()
			rev, err = p.parseRef()
		}

		if err != nil {
			return []revisioner{}, err
		}

		revs = append(revs, rev)
	}
}

// validateFullRevision ensures all revisioner chunks make a valid revision
func (p *parser) validateFullRevision(chunks *[]revisioner) error {
	var hasReference bool

	for i, chunk := range *chunks {
		switch chunk.(type) {
		case ref:
			if i == 0 {
				hasReference = true
			} else {
				return &ErrInvalidRevision{"reference must be defined once at the beginning"}
			}
		case atDate:
			if len(*chunks) == 1 || hasReference && len(*chunks) == 2 {
				return nil
			}

			return &ErrInvalidRevision{"@ statement is not valid, could be : <refname>@{<ISO-8601 date>}, @{<ISO-8601 date>}"}
		case atReflog:
			if len(*chunks) == 1 || hasReference && len(*chunks) == 2 {
				return nil
			}

			return &ErrInvalidRevision{"@ statement is not valid, could be : <refname>@{<n>}, @{<n>}"}
		case atCheckout:
			if len(*chunks) == 1 {
				return nil
			}

			return &ErrInvalidRevision{"@ statement is not valid, could be : @{-<n>}"}
		case atUpstream:
			if len(*chunks) == 1 || hasReference && len(*chunks) == 2 {
				return nil
			}

			return &ErrInvalidRevision{"@ statement is not valid, could be : <refname>@{upstream}, @{upstream}, <refname>@{u}, @{u}"}
		case atPush:
			if len(*chunks) == 1 || hasReference && len(*chunks) == 2 {
				return nil
			}

			return &ErrInvalidRevision{"@ statement is not valid, could be : <refname>@{push}, @{push}"}
		}
	}

	return nil
}

// parseAt extract @ statements
func (p *parser) parseAt() (revisioner, error) {
	var tok, nextTok token
	var lit, nextLit string

	tok, lit = p.scan()

	if tok != at {
		return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" found must be "@"`, lit)}
	}

	tok, lit = p.scan()

	if tok != obrace {
		p.unscan()

		return ref("HEAD"), nil
	}

	tok, lit = p.scan()
	nextTok, nextLit = p.scan()

	switch {
	case tok == word && (lit == "u" || lit == "upstream") && nextTok == cbrace:
		return atUpstream{}, nil
	case tok == word && lit == "push" && nextTok == cbrace:
		return atPush{}, nil
	case tok == number && nextTok == cbrace:
		n, _ := strconv.Atoi(lit)

		return atReflog{n}, nil
	case tok == minus && nextTok == number:
		n, _ := strconv.Atoi(nextLit)

		t, _ := p.scan()

		if t != cbrace {
			return nil, &ErrInvalidRevision{fmt.Sprintf(`missing "}" in @{-n} structure`)}
		}

		return atCheckout{n}, nil
	default:
		p.unscan()

		date := lit

		for {
			tok, lit = p.scan()

			switch {
			case tok == cbrace:
				t, err := time.Parse("2006-01-02T15:04:05Z", date)

				if err != nil {
					return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`wrong date "%s" must fit ISO-8601 format : 2006-01-02T15:04:05Z`, date)}
				}

				return atDate{t}, nil
			default:
				date += lit
			}
		}
	}
}

// parseTilde extract ~ statements
func (p *parser) parseTilde() (revisioner, error) {
	var tok token
	var lit string

	tok, lit = p.scan()

	if tok != tilde {
		return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" found must be "~"`, lit)}
	}

	tok, lit = p.scan()

	switch {
	case tok == number:
		n, _ := strconv.Atoi(lit)

		return tildePath{n}, nil
	case tok == tilde || tok == caret || tok == eof:
		p.unscan()
		return tildePath{1}, nil
	default:
		return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a valid revision suffix component`, lit)}
	}
}

// parseCaret extract ^ statements
func (p *parser) parseCaret() (revisioner, error) {
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

		r, err := p.parseCaretBraces()

		if err != nil {
			return (revisioner)(struct{}{}), err
		}

		return r, nil
	case tok == number:
		n, _ := strconv.Atoi(lit)

		return caretPath{n}, nil
	case tok == caret || tok == tilde || tok == eof:
		p.unscan()
		return caretPath{1}, nil
	default:
		return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`"%s" is not a valid revision suffix component`, lit)}
	}
}

// parseCaretBraces extract ^{<data>} statements
// todo : add regexp checker
func (p *parser) parseCaretBraces() (revisioner, error) {
	var tok, nextTok token
	var lit, _ string
	start := true
	reg := caretReg{}

	tok, lit = p.scan()

	if tok != obrace {
		return []revisioner{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" found must be "{" after ^`, lit)}
	}

	for {
		tok, lit = p.scan()
		nextTok, _ = p.scan()

		switch {
		case tok == word && nextTok == cbrace && (lit == "commit" || lit == "tree" || lit == "blob" || lit == "tag" || lit == "object"):
			return caretType{lit}, nil
		case reg.re == "" && tok == cbrace:
			return caretType{"tag"}, nil
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

// parseColon extract : statements
func (p *parser) parseColon() (revisioner, error) {
	var tok token
	var lit string

	tok, lit = p.scan()

	if tok != colon {
		return []revisioner{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" found must be ":"`, lit)}
	}

	tok, lit = p.scan()

	switch tok {
	case slash:
		p.unscan()
		return p.parseColonSlash()
	default:
		p.unscan()
		return p.parseColonDefault()
	}
}

// parseColonSlash extract :/<data> statements
// todo : add regexp checker
func (p *parser) parseColonSlash() (revisioner, error) {
	var tok, nextTok token
	var lit string
	reg := colonReg{}

	tok, lit = p.scan()

	if tok != slash {
		return []revisioner{}, &ErrInvalidRevision{fmt.Sprintf(`"%s" found must be "/"`, lit)}
	}

	for {
		tok, lit = p.scan()
		nextTok, _ = p.scan()

		switch {
		case tok == emark && nextTok == emark:
			reg.re += lit
		case reg.re == "" && tok == emark && nextTok == minus:
			reg.negate = true
		case reg.re == "" && tok == emark:
			return (revisioner)(struct{}{}), &ErrInvalidRevision{fmt.Sprintf(`revision suffix brace component sequences starting with "/!" others than those defined are reserved`)}
		case tok == eof:
			return reg, nil
		default:
			p.unscan()
			reg.re += lit
		}
	}
}

// parseColonDefault extract :<data> statements
func (p *parser) parseColonDefault() (revisioner, error) {
	var tok token
	var lit string
	var path string
	var stage int
	var n = -1

	tok, lit = p.scan()
	nextTok, _ := p.scan()

	if tok == number && nextTok == colon {
		n, _ = strconv.Atoi(lit)
	}

	switch n {
	case 0, 1, 2, 3:
		stage = n
	default:
		path += lit
		p.unscan()
	}

	for {
		tok, lit = p.scan()

		switch {
		case tok == eof:
			return colonPath{path, stage}, nil
		default:
			path += lit
		}
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
