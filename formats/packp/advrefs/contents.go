// Package advrefs implements encoding and decoding advertised-refs
// messages from a git-upload-pack command, as explained in
// https://github.com/git/git/blob/master/Documentation/technical/pack-protocol.txt.
package advrefs

import (
	"gopkg.in/src-d/go-git.v4/clients/common"
	"gopkg.in/src-d/go-git.v4/core"
)

const (
	hashSize = 40
	head     = "HEAD"
	noHead   = "capabilities^{}"
)

var (
	sp         = []byte(" ")
	null       = []byte("\x00")
	eol        = []byte("\n")
	peeled     = []byte("^{}")
	shallow    = []byte("shallow ")
	noHeadMark = []byte(" capabilities^{}\x00")
)

// Contents values represent the information transmitted on an
// advertised-refs message.  Values from this type are not zero-value
// safe, use the New function instead.
type AdvRefs struct {
	Head     *core.Hash
	Caps     *common.Capabilities
	Refs     map[string]core.Hash
	Peeled   map[string]core.Hash
	Shallows []core.Hash
}

// NewAdvRefs returns a pointer to a new AdvRefs value, ready to be used.
func NewAdvRefs() *AdvRefs {
	return &AdvRefs{
		Caps:     common.NewCapabilities(),
		Refs:     map[string]core.Hash{},
		Peeled:   map[string]core.Hash{},
		Shallows: []core.Hash{},
	}
}
