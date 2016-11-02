package memory

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4/utils/fs"
)

const separator = '/'

// Memory a very convenient filesystem based on memory files
type Memory struct {
	base string
	s    *storage
}

//New returns a new Memory filesystem
func New() *Memory {
	return &Memory{
		base: "/",
		s:    &storage{make(map[string]*file, 0)},
	}
}

func (fs *Memory) Create(filename string) (fs.File, error) {
	return fs.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func (fs *Memory) Open(filename string) (fs.File, error) {
	return fs.OpenFile(filename, os.O_RDONLY, 0)
}

func (fs *Memory) OpenFile(filename string, flag int, perm os.FileMode) (fs.File, error) {
	fullpath := fs.Join(fs.base, filename)
	f, ok := fs.s.files[fullpath]
	if !ok && flag&os.O_CREATE == 0 {
		return nil, os.ErrNotExist
	}

	if f == nil {
		fs.s.files[fullpath] = newFile(fs.base, fullpath, flag)
		return fs.s.files[fullpath], nil
	}

	n := newFile(fs.base, fullpath, flag)
	n.c = f.c

	if flag&os.O_APPEND != 0 {
		n.p = n.c.size
	}

	if flag&os.O_TRUNC != 0 {
		n.c.Truncate()
	}

	return n, nil
}

func (fs *Memory) Stat(filename string) (fs.FileInfo, error) {
	fullpath := fs.Join(fs.base, filename)

	if _, ok := fs.s.files[filename]; ok {
		return newFileInfo(fs.base, fullpath, fs.s.files[filename].c.size), nil
	}

	info, err := fs.ReadDir(filename)
	if err == nil && len(info) != 0 {
		return newFileInfo(fs.base, fullpath, int64(len(info))), nil
	}

	return nil, os.ErrNotExist
}

func (fs *Memory) ReadDir(base string) (entries []fs.FileInfo, err error) {
	base = fs.Join(fs.base, base)

	dirs := make(map[string]bool, 0)
	for fullpath, f := range fs.s.files {
		if !strings.HasPrefix(fullpath, base) {
			continue
		}

		fullpath, _ = filepath.Rel(base, fullpath)
		parts := strings.Split(fullpath, string(separator))

		if len(parts) != 1 {
			dirs[parts[0]] = true
			continue
		}

		entries = append(entries, newFileInfo(fs.base, fullpath, f.c.size))
	}

	for path := range dirs {
		entries = append(entries, &fileInfo{
			name:  path,
			isDir: true,
		})
	}

	return
}

func (fs *Memory) TempFile(dir, prefix string) (fs.File, error) {
	filename := fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	fullpath := fs.Join(fs.base, dir, filename)
	return fs.Create(fullpath)
}

func (fs *Memory) Rename(from, to string) error {
	from = fs.Join(fs.base, from)
	to = fs.Join(fs.base, to)

	if _, ok := fs.s.files[from]; !ok {
		return os.ErrNotExist
	}

	fs.s.files[to] = fs.s.files[from]
	fs.s.files[to].BaseFilename = to
	delete(fs.s.files, from)

	return nil
}

func (fs *Memory) Remove(filename string) error {
	if _, err := fs.Stat(filename); err != nil {
		return err
	}

	fullpath := fs.Join(fs.base, filename)
	delete(fs.s.files, fullpath)
	return nil
}

func (fs *Memory) clean(path string) string {
	if len(path) <= 1 {
		if path != string(separator) {
			return path
		}

		return ""
	}

	if path[0] == separator {
		path = path[1:]
	}

	l := len(path)
	if path[l-1] == separator {
		path = path[:l-1]
	}

	return path
}

func (fs *Memory) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (fs *Memory) Dir(path string) fs.Filesystem {
	return &Memory{
		base: fs.Join(fs.base, path),
		s:    fs.s,
	}
}

func (fs *Memory) Base() string {
	return fs.base
}

type file struct {
	fs.BaseFile

	c     *content
	p     int64
	flags int
}

func newFile(base, fullpath string, flags int) *file {
	filename, _ := filepath.Rel(base, fullpath)

	return &file{
		BaseFile: fs.BaseFile{BaseFilename: filename},
		c:        &content{},
		flags:    flags,
	}
}

func (f *file) Read(b []byte) (int, error) {
	if f.IsClosed() {
		return 0, fs.ErrClosed
	}

	if f.flags&os.O_RDWR != 0 && f.flags&os.O_RDONLY != 0 {
		return 0, errors.New("read not supported")
	}

	n, err := f.c.ReadAt(b, f.p)
	f.p += int64(n)

	return n, err
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		if offset == 0 {
			return f.p, nil
		}

		f.p += offset
	case io.SeekStart:
		f.p = offset
	case io.SeekEnd:
		f.p = f.c.size - offset
	}

	return f.p, nil
}

func (f *file) Write(p []byte) (int, error) {
	if f.IsClosed() {
		return 0, fs.ErrClosed
	}

	if f.flags&os.O_RDWR != 0 && f.flags&os.O_WRONLY != 0 {
		return 0, errors.New("read not supported")
	}

	n, err := f.c.WriteAt(p, f.p)
	f.p += int64(n)

	return n, err
}

func (f *file) Close() error {
	f.Closed = true
	return nil
}

func (f *file) Open() error {
	f.Closed = false
	return nil
}

type fileInfo struct {
	name  string
	size  int64
	isDir bool
}

func newFileInfo(base, fullpath string, size int64) *fileInfo {
	filename, _ := filepath.Rel(base, fullpath)

	return &fileInfo{
		name: filename,
		size: size,
	}
}

func (fi *fileInfo) Name() string {
	return fi.name
}

func (fi *fileInfo) Size() int64 {
	return fi.size
}

func (fi *fileInfo) Mode() os.FileMode {
	return os.FileMode(0)
}

func (*fileInfo) ModTime() time.Time {
	return time.Now()
}

func (fi *fileInfo) IsDir() bool {
	return fi.isDir
}

func (*fileInfo) Sys() interface{} {
	return nil
}

type storage struct {
	files map[string]*file
}

type content struct {
	bytes []byte
	size  int64
}

func (c *content) WriteAt(p []byte, off int64) (int, error) {
	l := len(p)
	if int(off)+l > len(c.bytes) {
		buf := make([]byte, 2*len(c.bytes)+l)
		copy(buf, c.bytes)
		c.bytes = buf
	}

	n := copy(c.bytes[off:], p)
	if off+int64(n) > c.size {
		c.size = off + int64(n)
	}

	return n, nil
}

func (c *content) ReadAt(b []byte, off int64) (int, error) {
	if off >= c.size {
		return 0, io.EOF
	}

	l := int64(len(b))
	if off+l > c.size {
		l = c.size - off
	}

	n := copy(b, c.bytes[off:l])
	return n, nil
}

func (c *content) Truncate() {
	c.bytes = []byte{}
	c.size = 0
}
