package packfile

import "gopkg.in/src-d/go-git.v3/core"

type Reader interface {
	Read(p []byte) (int, error)
	ReadByte() (byte, error)
	Offset() (int64, error)
	RecallByOffset(int64) (core.Object, error)
	RecallByHash(core.Hash) (core.Object, error)
}
