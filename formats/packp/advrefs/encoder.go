package advrefs

import (
	"fmt"
	"io"
	"sort"

	"gopkg.in/src-d/go-git.v4/core"
	"gopkg.in/src-d/go-git.v4/formats/packp/pktline"
)

// The state of the encoder.  The provided Contents is used as input,
// then the run method goes over every field, adding the corresponding
// payloads that will get turned into real pkt-lines and made accesible
// through the ret field.
type Encoder struct {
	data *Contents // data to encode
	w    io.Writer // where to write the encoded data
	err  error     // sticky error
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

// Writes the advrefs contents, as pkt-lines, to the writer in e.
func (e *Encoder) Encode(data *Contents) error {
	e.data = data
	for state := encodeFirstLine; state != nil; {
		state = state(e)
	}

	return e.err
}

type encoderStateFn func(*Encoder) encoderStateFn

// Adds the first pkt-line payload: head hash, head ref and capabilities.
// Also handle the special case when no HEAD ref is found.
func encodeFirstLine(e *Encoder) encoderStateFn {
	var hash string
	var headRef string
	if e.data.Head == nil {
		hash = core.ZeroHash.String()
		headRef = noHead
	} else {
		hash = e.data.Head.String()
		headRef = head
	}

	var caps string
	if e.data.Caps != nil {
		e.data.Caps.Sort()
		caps = e.data.Caps.String()
	}

	payload := fmt.Sprintf("%s %s\x00%s\n", hash, string(headRef), caps)
	var p pktline.PktLine
	p, e.err = pktline.NewFromStrings(payload)
	if e.err != nil {
		return nil
	}

	if _, e.err = io.Copy(e.w, p); e.err != nil {
		return nil
	}

	return encodeRefs
}

// adds the (sorted) refs: hash SP refname EOL
// and their peeled refs if any.
func encodeRefs(e *Encoder) encoderStateFn {
	refs := sortRefs(e.data.Refs)
	for _, r := range refs {
		h, _ := e.data.Refs[r]
		hash := h.String()
		n := len(hash) + len(sp) + len(r) + len(eol)
		payload := make([]byte, n)
		nw := copy(payload, hash)
		nw += copy(payload[nw:], sp)
		nw += copy(payload[nw:], r)
		nw += copy(payload[nw:], eol)
		var p pktline.PktLine
		p, e.err = pktline.New(payload)
		if e.err != nil {
			return nil
		}
		if _, e.err = io.Copy(e.w, p); e.err != nil {
			return nil
		}

		if h, ok := e.data.Peeled[r]; ok {
			hash = h.String()
			n := len(hash) + len(sp) + len(r) + len(peeled) + len(eol)
			payload := make([]byte, n)
			nw := copy(payload, hash)
			nw += copy(payload[nw:], sp)
			nw += copy(payload[nw:], r)
			nw += copy(payload[nw:], peeled)
			nw += copy(payload[nw:], eol)
			var p pktline.PktLine
			p, e.err = pktline.New(payload)
			if e.err != nil {
				return nil
			}
			if _, e.err = io.Copy(e.w, p); e.err != nil {
				return nil
			}
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
func encodeShallow(e *Encoder) encoderStateFn {
	sorted := sortShallows(e.data.Shallows)
	for _, hash := range sorted {
		n := len(shallow) + len(hash) + len(eol)
		payload := make([]byte, n)
		nw := copy(payload, shallow)
		nw += copy(payload[nw:], hash)
		nw += copy(payload[nw:], eol)
		var p pktline.PktLine
		p, e.err = pktline.New(payload)
		if e.err != nil {
			return nil
		}
		if _, e.err = io.Copy(e.w, p); e.err != nil {
			return nil
		}
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

func encodeFlush(e *Encoder) encoderStateFn {
	p, _ := pktline.New([]byte{})
	if _, e.err = io.Copy(e.w, p); e.err != nil {
		return nil
	}

	return nil
}
