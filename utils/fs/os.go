package fs

import (
	"os"
	"path/filepath"
)

type OS struct {
}

func NewOS() *OS {
	return &OS{}
}

func (o *OS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (o *OS) Open(path string) (ReadSeekCloser, error) {
	return os.Open(path)
}

func (o *OS) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (o *OS) Glob(p string) ([]string, error) {
	return filepath.Glob(p)
}

func (o *OS) Rel(base, target string) (string, error) {
	return filepath.Rel(base, target)
}
