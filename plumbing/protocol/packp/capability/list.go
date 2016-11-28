package capability

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrUnknownCapability is returned if a unknown capability is given
	ErrUnknownCapability = errors.New("unknown capability")
	// ErrArgumentsRequired is returned if no arguments are giving with a
	// capability that requires arguments
	ErrArgumentsRequired = errors.New("arguments required")
	// ErrArguments is returned if arguments are given with a capabilities that
	// not supports arguments
	ErrArguments = errors.New("arguments not allowed")
	// ErrEmtpyArgument is returned when an empty value is given
	ErrEmtpyArgument = errors.New("empty argument")
	// ErrMultipleArguments multiple argument given to a capabilities that not
	// support it
	ErrMultipleArguments = errors.New("multiple arguments not allowed")
)

// List represents a list of capabilities
type List struct {
	m map[Capability]*Entry
	o []string
}

// Entry represents a server capability
type Entry struct {
	// Name of the capability
	Name Capability
	// Values, values of the capability, only a few capabilities supports values
	Values []string
}

// NewList returns a new List of capabilities
func NewList() *List {
	return &List{
		m: make(map[Capability]*Entry),
	}
}

// IsEmpty returns true if the List is empty
func (l *List) IsEmpty() bool {
	return len(l.o) == 0
}

// Decode decodes list of capabilities from raw into the list
func (l *List) Decode(raw []byte) error {
	for _, data := range bytes.Split(raw, []byte{' '}) {
		pair := bytes.SplitN(data, []byte{'='}, 2)

		c := Capability(pair[0])
		if len(pair) == 1 {
			if err := l.Add(c); err != nil {
				return err
			}

			continue
		}

		if err := l.Add(c, string(pair[1])); err != nil {
			return err
		}
	}

	return nil
}

// Get returns the values for a capability
func (l *List) Get(capability Capability) *Entry {
	return l.m[capability]
}

// Set sets a capability removing the previous values
func (l *List) Set(capability Capability, values ...string) error {
	if _, ok := l.m[capability]; ok {
		delete(l.m, capability)
	}

	return l.Add(capability, values...)
}

// Add adds a capability, values are optional
func (l *List) Add(c Capability, values ...string) error {
	if err := l.validate(c, values); err != nil {
		return err
	}

	if !l.Supports(c) {
		l.m[c] = &Entry{Name: c}
		l.o = append(l.o, c.String())
	}

	if len(values) == 0 {
		return nil
	}

	if !multipleArgument[c] && len(l.m[c].Values) > 0 {
		return ErrMultipleArguments
	}

	l.m[c].Values = append(l.m[c].Values, values...)
	return nil
}

func (l *List) validate(c Capability, values []string) error {
	if _, ok := valid[c]; !ok {
		return ErrUnknownCapability
	}

	if requiresArgument[c] && len(values) == 0 {
		return ErrArgumentsRequired
	}

	if !requiresArgument[c] && len(values) != 0 {
		return ErrArguments
	}

	if !multipleArgument[c] && len(values) > 1 {
		return ErrMultipleArguments
	}

	for _, v := range values {
		if v == "" {
			return ErrEmtpyArgument
		}
	}

	return nil
}

// Supports returns true if capability is present
func (l *List) Supports(capability Capability) bool {
	_, ok := l.m[capability]
	return ok
}

func (l *List) String() string {
	sort.Strings(l.o)

	var o string
	for _, key := range l.o {
		cap := l.m[Capability(key)]
		if len(cap.Values) == 0 {
			o += key + " "
			continue
		}

		for _, value := range cap.Values {
			o += fmt.Sprintf("%s=%s ", key, value)
		}
	}

	return strings.Trim(o, " ")
}
