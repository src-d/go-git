package gitdir

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v3/clients/common"
	"gopkg.in/src-d/go-git.v3/core"
)

const (
	suffix         = ".git"
	packedRefsPath = "packed-refs"
)

var (
	// ErrBadGitDirName is returned when the passed path is not a .git directory.
	ErrBadGitDirName = errors.New(`Bad git dir name (must end in ".git")`)
	// ErrIdxNotFound is returned when the idx file is not found on the repository.
	ErrIdxNotFound = errors.New("idx file not found")
)

// The Dir type represents a local git repository on disk. This
// type is not zero-value-safe, use the New function to initialize it.
type Dir struct {
	path string
	refs map[string]core.Hash
}

// New returns a Dir value ready to be used. The path argument must be
// an existing git repository directory (e.g. "/foo/bar/.git").
func New(path string) (*Dir, error) {
	d := &Dir{}
	var err error

	d.path, err = cleanPath(path)
	if err != nil {
		return nil, err
	}

	if d.isInvalidPath() {
		return nil, ErrBadGitDirName
	}

	return d, nil
}

func cleanPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return filepath.Clean(abs), nil
}

func (d *Dir) isInvalidPath() bool {
	return !strings.HasSuffix(d.path, suffix)
}

// Refs scans the git directory collecting references, which it returns.
// Symbolic references are resolved and included in the output.
func (d *Dir) Refs() (map[string]core.Hash, error) {
	var err error

	if err = d.initRefsFromPackedRefs(); err != nil {
		return nil, err
	}

	if err = d.addRefsFromRefDir(); err != nil {
		return nil, err
	}

	return d.refs, err
}

// Capabilities scans the git directory collection capabilities, which it returns.
func (d *Dir) Capabilities() (*common.Capabilities, error) {
	c := common.NewCapabilities()

	err := d.addSymRefCapability(c)

	return c, err
}

func (d *Dir) addSymRefCapability(cap *common.Capabilities) (err error) {
	f, err := os.Open(filepath.Join(d.path, "HEAD"))
	if err != nil {
		return err
	}

	defer func() {
		errClose := f.Close()
		if err == nil {
			err = errClose
		}
	}()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	contents := strings.TrimSpace(string(b))

	c := "symref"
	ref := strings.TrimPrefix(contents, symRefPrefix)
	cap.Set(c, "HEAD:"+ref)

	return nil
}

// ReadSeekCloser is an io.ReadSeeker with a Close method.
type ReadSeekCloser interface {
	io.ReadSeeker
	Close() error
}

// Packfile returns the path of the packfile in the repository.
func (d *Dir) Packfile() (string, error) {
	p := d.pattern(true)

	l, err := filepath.Glob(p)
	if err != nil {
		return "", err
	}

	if len(l) == 0 {
		return "", fmt.Errorf("packfile not found")
	}

	if len(l) > 1 {
		return "", fmt.Errorf("found more than one packfile")
	}

	return l[0], nil
}

func (d *Dir) pattern(isPackfile bool) string {
	// packfile pattern: d.path + /objects/pack/pack-40hexs.pack
	//      idx pattern: d.path + /objects/pack/pack-40hexs.idx
	base := filepath.Join(d.path, "objects")
	base = filepath.Join(base, "pack")
	file := filePattern + extension(isPackfile)
	return filepath.Join(base, file)
}

func extension(isPackfile bool) string {
	if isPackfile {
		return ".pack"
	}

	return ".idx"
}

// "pack-" followed by 40 chars representing hexadecimal numbers
const filePattern = "pack-[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]"

// Idxfile returns the path of the idx file in the repository.
func (d *Dir) Idxfile() (string, error) {
	p := d.pattern(false)

	l, err := filepath.Glob(p)
	if err != nil {
		return "", err
	}

	if len(l) == 0 {
		return "", ErrIdxNotFound
	}

	if len(l) > 1 {
		return "", fmt.Errorf("found more than one idxfile")
	}

	return l[0], nil
}
