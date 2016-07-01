package fs

import (
	"io"
	"os"
)

type FS interface {
	Stat(path string) (os.FileInfo, error)
	Open(path string) (ReadSeekCloser, error)
	Join(elem ...string) string
	Glob(p string) ([]string, error)
	Rel(string, string) (string, error)
}

type ReadSeekCloser interface {
	io.ReadCloser
	io.Seeker
}
