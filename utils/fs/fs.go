package fs

import (
	"io"
	"os"
)

type FS interface {
	Stat(path string) (os.FileInfo, error)
	Open(path string) (ReadSeekCloser, error)
	ReadDir(path string) ([]os.FileInfo, error)
	Join(elem ...string) string
}

type ReadSeekCloser interface {
	io.ReadCloser
	io.Seeker
}
