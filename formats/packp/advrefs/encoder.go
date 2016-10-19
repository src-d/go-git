package advrefs

import (
	"fmt"
	"io"
	"sort"

	"gopkg.in/src-d/go-git.v4/clients/common"
	"gopkg.in/src-d/go-git.v4/core"
	"gopkg.in/src-d/go-git.v4/formats/packp/pktline"
)

// An Encoder writes AdvRefs values to an output stream.
type Encoder struct {
	data *AdvRefs  // data to encode
	w    io.Writer // where to write the encoded data
	err  error     // sticky error
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

// Encode writes the AdvRefs encoding of v to the stream.
//
// All the payloads will end with a newline character.  Capabilities,
// references and shallows are writen in alphabetical order, except for
// peeled references that always follow their corresponding references.
func (e *Encoder) Encode(v *AdvRefs) error {
	e.data = v

	for state := encodeFirstLine; state != nil; {
		state = state(e)
	}

	return e.err
}

type encoderStateFn func(*Encoder) encoderStateFn

// Formats a payload using the default formats for its operands and
// write the corresponding pktline to the encoder writer.
func (e *Encoder) writePktLine(format string, a ...interface{}) error {
	payload := fmt.Sprintf(format, a...)

	p, err := pktline.NewFromStrings(payload)
	if err != nil {
		return err
	}

	if _, err = io.Copy(e.w, p); err != nil {
		return err
	}

	return nil
}

// Adds the first pkt-line payload: head hash, head ref and capabilities.
// Also handle the special case when no HEAD ref is found.
func encodeFirstLine(e *Encoder) encoderStateFn {
	head := formatHead(e.data.Head)
	sep := formatSeparator(e.data.Head)
	caps := formatCaps(e.data.Caps)

	if e.err = e.writePktLine("%s %s\x00%s\n", head, sep, caps); e.err != nil {
		return nil
	}

	return encodeRefs
}

func formatHead(h *core.Hash) string {
	if h == nil {
		return core.ZeroHash.String()
	}

	return h.String()
}

func formatSeparator(h *core.Hash) string {
	if h == nil {
		return noHead
	}

	return head
}

func formatCaps(c *common.Capabilities) string {
	if c == nil {
		return ""
	}

	c.Sort()

	return c.String()
}

// Adds the (sorted) refs: hash SP refname EOL
// and their peeled refs if any.
func encodeRefs(e *Encoder) encoderStateFn {
	refs := sortRefs(e.data.Refs)
	for _, r := range refs {
		hash, _ := e.data.Refs[r]
		if e.err = e.writePktLine("%s %s\n", hash.String(), r); e.err != nil {
			return nil
		}

		if hash, ok := e.data.Peeled[r]; ok {
			if e.err = e.writePktLine("%s %s^{}\n", hash.String(), r); e.err != nil {
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

// Adds the (sorted) shallows: "shallow" SP hash EOL
func encodeShallow(e *Encoder) encoderStateFn {
	sorted := sortShallows(e.data.Shallows)
	for _, hash := range sorted {
		if e.err = e.writePktLine("shallow %s\n", hash); e.err != nil {
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
