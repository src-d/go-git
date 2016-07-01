package fs

import (
	"io/ioutil"
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

func (o *OS) ReadDir(path string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(path)
}

func (o *OS) Join(elem ...string) string {
	return filepath.Join(elem...)
}
