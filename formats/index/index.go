package index

import (
	"time"

	"gopkg.in/src-d/go-git.v3/core"
)

// An Index represents an idx file in memory.
type Index struct {
	Version    uint32
	EntryCount uint32
	Entries    []Entry
}

type Entry struct {
	CreatedAt  time.Time
	ModifiedAt time.Time
	Dev, Inode uint32
	Mode       uint32
	UID, GID   uint32
	Size       uint32
	Flags      uint16
	Hash       core.Hash
	Name       string
}
