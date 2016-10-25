package config

import (
	"fmt"
	"io"
	"io/ioutil"

	"errors"
	"gopkg.in/gcfg.v1/scanner"
	"gopkg.in/gcfg.v1/token"
	"gopkg.in/warnings.v0"
)

// Since git v1.8.1-rc1 config, if there are multiple
// definitions of a key, the last one wins.
// http://article.gmane.org/gmane.linux.kernel/1407184

// A Decoder reads and decodes config files from an input stream.
type Decoder struct {
	io.Reader
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r}
}

// Decode reads the whole config from its input and stores it in the
// value pointed to by config.
func (d *Decoder) Decode(config *Config) error {
	src, err := ioutil.ReadAll(d)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	c := warnings.NewCollector(func(error) bool {
		return true
	})

	err = decode(c, config, fset, file, src)
	if err != nil {
		return err
	}

	return nil
}

// Based on https://github.com/go-gcfg/gcfg/blob/5b9f94ee80b2331c3982477bd84be8edd857df33/read.go#L51
func decode(c *warnings.Collector, config *Config, fset *token.FileSet,
	file *token.File, src []byte) error {

	var err error
	var s scanner.Scanner
	var errs scanner.ErrorList
	s.Init(file, src, func(p token.Position, m string) { errs.Add(p, m) }, 0)
	sect, sectsub := "", ""
	pos, tok, lit := s.Scan()
	errfn := func(msg string) error {
		return fmt.Errorf("%s: %s", fset.Position(pos), msg)
	}
	for {
		if errs.Len() > 0 {
			if err := c.Collect(errs.Err()); err != nil {
				return err
			}
		}
		switch tok {
		case token.EOF:
			return nil
		case token.EOL, token.COMMENT:
			pos, tok, lit = s.Scan()
		case token.LBRACK:
			pos, tok, lit = s.Scan()
			if errs.Len() > 0 {
				if err := c.Collect(errs.Err()); err != nil {
					return err
				}
			}
			if tok != token.IDENT {
				if err := c.Collect(errfn("expected section name")); err != nil {
					return err
				}
			}
			sect, sectsub = lit, ""
			pos, tok, lit = s.Scan()
			if errs.Len() > 0 {
				if err := c.Collect(errs.Err()); err != nil {
					return err
				}
			}
			if tok == token.STRING {
				sectsub, err = unquote(lit)
				if err != nil {
					if err := c.Collect(err); err != nil {
						return err
					}
				}
				if sectsub == "" {
					if err := c.Collect(errfn("empty subsection name")); err != nil {
						return err
					}
				}
				pos, tok, lit = s.Scan()
				if errs.Len() > 0 {
					if err := c.Collect(errs.Err()); err != nil {
						return err
					}
				}
			}
			if tok != token.RBRACK {
				if sectsub == "" {
					if err := c.Collect(errfn("expected subsection name or right bracket")); err != nil {
						return err
					}
				}
				if err := c.Collect(errfn("expected right bracket")); err != nil {
					return err
				}
			}
			pos, tok, lit = s.Scan()
			if tok != token.EOL && tok != token.EOF && tok != token.COMMENT {
				if err := c.Collect(errfn("expected EOL, EOF, or comment")); err != nil {
					return err
				}
			}
			// If a section/subsection header was found, ensure a
			// container object is created, even if there are no
			// variables further down.
			config.Section(sect)
			if sectsub != "" {
				config.Section(sect).Subsection(sectsub)
			}
		case token.IDENT:
			if sect == "" {
				if err := c.Collect(errfn("expected section header")); err != nil {
					return err
				}
			}
			n := lit
			pos, tok, lit = s.Scan()
			if errs.Len() > 0 {
				return errs.Err()
			}
			blank, v := tok == token.EOF || tok == token.EOL || tok == token.COMMENT, ""
			if !blank {
				if tok != token.ASSIGN {
					if err := c.Collect(errfn("expected '='")); err != nil {
						return err
					}
				}
				pos, tok, lit = s.Scan()
				if errs.Len() > 0 {
					if err := c.Collect(errs.Err()); err != nil {
						return err
					}
				}
				if tok != token.STRING {
					if err := c.Collect(errfn("expected value")); err != nil {
						return err
					}
				}
				v, err = unquote(lit)
				if err != nil {
					if err := c.Collect(err); err != nil {
						return err
					}
				}
				pos, tok, lit = s.Scan()
				if errs.Len() > 0 {
					if err := c.Collect(errs.Err()); err != nil {
						return err
					}
				}
				if tok != token.EOL && tok != token.EOF && tok != token.COMMENT {
					if err := c.Collect(errfn("expected EOL, EOF, or comment")); err != nil {
						return err
					}
				}
			}
			config.AddOption(sect, sectsub, n, v)
		default:
			if sect == "" {
				if err := c.Collect(errfn("expected section header")); err != nil {
					return err
				}
			}
			if err := c.Collect(errfn("expected section header or variable declaration")); err != nil {
				return err
			}
		}
	}
	panic("never reached")
}

// https://github.com/go-gcfg/gcfg/blob/5b9f94ee80b2331c3982477bd84be8edd857df33/read.go#L15
var unescape = map[rune]rune{'\\': '\\', '"': '"', 'n': '\n', 't': '\t'}

// no error: invalid literals should be caught by scanner
// https://github.com/go-gcfg/gcfg/blob/5b9f94ee80b2331c3982477bd84be8edd857df33/read.go#L18
func unquote(s string) (string, error) {
	u, q, esc := make([]rune, 0, len(s)), false, false
	for _, c := range s {
		if esc {
			uc, ok := unescape[c]
			switch {
			case ok:
				u = append(u, uc)
				fallthrough
			case !q && c == '\n':
				esc = false
				continue
			}
			return "", errors.New("invalid escape sequence")
		}
		switch c {
		case '"':
			q = !q
		case '\\':
			esc = true
		default:
			u = append(u, c)
		}
	}
	if q {
		return "", errors.New("missing end quote")
	}
	if esc {
		return "", errors.New("invalid escape sequence")
	}
	return string(u), nil
}
