package advrefs

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v4/core"
	"gopkg.in/src-d/go-git.v4/formats/packp/pktline"
)

// The state of the parser: items are detected from the scanner s and
// get decoded and stored in the corresponding sections of ret.
type Decoder struct {
	s     *pktline.Scanner // a pkt-line scanner from the output of a git-upload-pack command
	line  []byte           // current pkt-line contents, use parser.nextLine() to make it advance
	nLine int              // current pkt-line number for debugging, begins at 1
	hash  core.Hash        // last hash read
	err   error            // sticky error, use the parser.error() method to fill this out
	ret   *AdvRefs         // parsed data
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		s: pktline.NewScanner(r),
	}
}

func (d *Decoder) Decode(ar *AdvRefs) error {
	d.ret = ar

	for state := decodeFirstHash; state != nil; {
		state = state(d)
	}

	return d.err
}

type decoderStateFn func(*Decoder) decoderStateFn

// fills out the parser stiky error
func (p *Decoder) error(format string, a ...interface{}) {
	p.err = fmt.Errorf("pkt-line %d: %s", p.nLine,
		fmt.Sprintf(format, a...))
}

// Reads a new pkt-line from the scanner, copies its payload into p.line
// and increments p.nLine.  A successful invocation returns true,
// otherwise, false is returned and the sticky error is filled out
// accordingly.  Trims eols at the end of the payloads.
func (p *Decoder) nextLine() bool {
	p.nLine++

	if !p.s.Scan() {
		if err := p.s.Err(); err != nil {
			p.error("%s", err)
			return false
		}
		p.error("EOF")

		return false
	}

	p.line = p.s.Bytes()
	p.line = bytes.TrimSuffix(p.line, eol)

	return true
}

// If the first hash is zero, then a no-refs is comming. Otherwise, a
// list-of-refs is comming, and the hash will be followed by the first
// advertised ref.
func decodeFirstHash(p *Decoder) decoderStateFn {
	if ok := p.nextLine(); !ok {
		return nil
	}

	if len(p.line) < hashSize {
		p.error("cannot read hash, pkt-line too short")
		return nil
	}

	if _, err := hex.Decode(p.hash[:], p.line[:hashSize]); err != nil {
		p.error("invalid hash text: %s", err)
		return nil
	}

	p.line = p.line[hashSize:]

	if p.hash.IsZero() {
		return decodeSkipNoRefs
	}

	return decodeFirstRef
}

// skips SP "capabilities^{}" NUL
func decodeSkipNoRefs(p *Decoder) decoderStateFn {
	if len(p.line) < len(noHeadMark) {
		p.error("too short zero-id ref")
		return nil
	}

	if !bytes.HasPrefix(p.line, noHeadMark) {
		p.error("malformed zero-id ref")
		return nil
	}

	p.line = p.line[len(noHeadMark):]

	return decodeCaps
}

// SP refname NULL
func decodeFirstRef(l *Decoder) decoderStateFn {
	if len(l.line) < 3 {
		l.error("line too short after hash")
		return nil
	}

	if !bytes.HasPrefix(l.line, sp) {
		l.error("no space after hash")
		return nil
	}
	l.line = l.line[1:]

	chunks := bytes.SplitN(l.line, null, 2)
	if len(chunks) < 2 {
		l.error("NULL not found")
		return nil
	}
	ref := chunks[0]
	l.line = chunks[1]

	if bytes.Equal(ref, []byte(head)) {
		l.ret.Head = &l.hash
	} else {
		l.ret.Refs[string(ref)] = l.hash
	}

	return decodeCaps
}

func decodeCaps(p *Decoder) decoderStateFn {
	if len(p.line) == 0 {
		return decodeOtherRefs
	}

	for _, c := range bytes.Split(p.line, sp) {
		name, values := readCapability(c)
		p.ret.Caps.Add(name, values...)
	}

	return decodeOtherRefs
}

// capabilities are a single string or a name=value.
// Even though we are only going to read at moust 1 value, we return
// a slice of values, as Capability.Add receives that.
func readCapability(data []byte) (name string, values []string) {
	pair := bytes.SplitN(data, []byte{'='}, 2)
	if len(pair) == 2 {
		values = append(values, string(pair[1]))
	}

	return string(pair[0]), values
}

// the refs are either tips (obj-id SP refname) or a peeled (obj-id SP refname^{}).
// If there are no refs, then there might be a shallow or flush-ptk.
func decodeOtherRefs(p *Decoder) decoderStateFn {
	if ok := p.nextLine(); !ok {
		return nil
	}

	if bytes.HasPrefix(p.line, shallow) {
		p.line = bytes.TrimPrefix(p.line, shallow)
		return decodeHalfReadShallow
	}

	if len(p.line) == 0 {
		return nil
	}

	saveTo := p.ret.Refs
	if bytes.HasSuffix(p.line, peeled) {
		p.line = bytes.TrimSuffix(p.line, peeled)
		saveTo = p.ret.Peeled
	}

	ref, hash, err := readRef(p.line)
	if err != nil {
		p.error("%s", err)
		return nil
	}
	saveTo[ref] = hash

	return decodeOtherRefs
}

// reads a ref-name
func readRef(data []byte) (string, core.Hash, error) {
	chunks := bytes.Split(data, sp)
	if len(chunks) == 1 {
		return "", core.ZeroHash, fmt.Errorf("malformed ref data: no space was found")
	}
	if len(chunks) > 2 {
		return "", core.ZeroHash, fmt.Errorf("malformed ref data: more than one space found")
	}

	return string(chunks[1]), core.NewHash(string(chunks[0])), nil
}

// reads a hash from a shallow pkt-line
func decodeHalfReadShallow(p *Decoder) decoderStateFn {
	if len(p.line) != hashSize {
		p.error(fmt.Sprintf(
			"malformed shallow hash: wrong length, expected 40 bytes, read %d bytes",
			len(p.line)))
		return nil
	}

	text := p.line[:hashSize]
	var h core.Hash
	if _, err := hex.Decode(h[:], text); err != nil {
		p.error("invalid hash text: %s", err)
		return nil
	}

	p.ret.Shallows = append(p.ret.Shallows, h)

	return decodeShallow
}

// keeps reading shallows until a flush-pkt is found
func decodeShallow(p *Decoder) decoderStateFn {
	if ok := p.nextLine(); !ok {
		return nil
	}

	if len(p.line) == 0 {
		return nil // succesfull parse of the advertised-refs message
	}

	if !bytes.HasPrefix(p.line, shallow) {
		p.error("malformed shallow prefix")
		return nil
	}
	p.line = bytes.TrimPrefix(p.line, shallow)

	return decodeHalfReadShallow
}
