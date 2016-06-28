package packfile

import "gopkg.in/src-d/go-git.v3/core"

var (
	// ErrDulicatedObject is returned by Remember if an object appears several
	// times in a packfile.
	ErrDuplicatedObj = NewError("duplicated object")
	// ErrRecall is returned by RecallByOffset or RecallByHash if the object
	// to recall cannot be returned.
	ErrRecall = NewError("cannot recall object")
)

// The Reader interface has all the functions needed by a packfile Parser to operate.
// resons for ReadByte:
// https://github.com/golang/go/commit/7ba54d45732219af86bde9a5b73c145db82b70c6
// https://groups.google.com/forum/#!topic/golang-nuts/fWTRdHpt0QI
// https://gowalker.org/compress/zlib#NewReader
type Reader interface {
	Read(p []byte) (int, error)
	ReadByte() (byte, error)
	Offset() (int64, error)
	Remember(int64, core.Object) error
	RecallByOffset(int64) (core.Object, error)
	RecallByHash(core.Hash) (core.Object, error)
}
