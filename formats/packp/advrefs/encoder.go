package advrefs

import (
	"io"
	"sort"

	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/packp/pktline"
)

// The state of the encoder.  The provided Contents is used as input,
// then the run method goes over every field, adding the corresponding
// payloads that will get turned into real pkt-lines and made accesible
// through the ret field.
type encoder struct {
	contents *Contents // input data
	payloads [][]byte  // the payloads that will be used for each pkt-line
	err      error     // sticky error
	ret      io.Reader // the whole advertised refs message (a sequence of pkt-lines)
}

func newEncoder(ar *Contents) *encoder {
	return &encoder{
		contents: ar,
		payloads: [][]byte{},
	}
}

// Encodes the contents into the corresponding pkt-lines.
func (e *encoder) run() (io.Reader, error) {
	for state := encodeFirstLine; state != nil; {
		state = state(e)
	}

	if e.err != nil {
		return nil, e.err
	}

	return e.ret, nil
}

type encoderStateFn func(*encoder) encoderStateFn

func (e *encoder) addPayload(bb ...[]byte) {
	p := []byte{}
	for _, b := range bb {
		p = append(p, b...)
	}
	e.payloads = append(e.payloads, p)
}

// Adds the first pkt-line payload: head hash, head ref and capabilities.
// Also handle the special case when no HEAD ref is found.
func encodeFirstLine(e *encoder) encoderStateFn {
	caps := []byte{}
	if e.contents.Caps != nil {
		e.contents.Caps.Sort()
		caps = []byte(e.contents.Caps.String())
	}

	if e.contents.Head == nil {
		hash := []byte(core.ZeroHash.String())
		e.addPayload(hash, noRefText, caps, eol)

		return encodeRefs
	}

	hash := []byte(e.contents.Head.String())
	e.addPayload(hash, sp, head, null, caps, eol)

	return encodeRefs
}

// adds the (sorted) refs: hash SP refname EOL
// and their peeled refs if any.
func encodeRefs(e *encoder) encoderStateFn {
	refs := sortRefs(e.contents.Refs)
	for _, r := range refs {
		h, _ := e.contents.Refs[r]
		hash := []byte(h.String())
		refname := []byte(r)
		e.addPayload(hash, sp, refname, eol)

		if h, ok := e.contents.Peeled[r]; ok {
			hash = []byte(h.String())
			refname = []byte(r)
			e.addPayload(hash, sp, refname, peeled, eol)
		}
	}

	return encodeShallow
}

func sortRefs(m map[string]core.Hash) []string {
	ret := make([]string, 0, len(m))
	for k := range m {
		ret = append(ret, k)
	}
	sort.Strings(ret)

	return ret
}

// adds the (sorted) shallows: "shallow" SP hash EOL
func encodeShallow(e *encoder) encoderStateFn {
	sorted := sortShallows(e.contents.Shallows)
	for _, hash := range sorted {
		e.addPayload(shallow, []byte(hash), eol)
	}

	return encodeFlush
}

func sortShallows(c []core.Hash) []string {
	ret := []string{}
	for _, h := range c {
		ret = append(ret, h.String())
	}
	sort.Strings(ret)

	return ret
}

func encodeFlush(e *encoder) encoderStateFn {
	e.addPayload([]byte{})

	return encodePayloads
}

// encode all payloads as pkt-lines
func encodePayloads(e *encoder) encoderStateFn {
	e.ret, e.err = pktline.New(e.payloads...)

	return nil
}
