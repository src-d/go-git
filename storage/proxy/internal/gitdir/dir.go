package gitdir

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/src-d/go-git.v3/clients/common"
	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/utils/fs"
)

const (
	suffix         = ".git"
	packedRefsPath = "packed-refs"
)

var (
	// ErrNotFound is returned by New when the path is not found.
	ErrNotFound = errors.New("path not found")
	// ErrIdxNotFound is returned by Idxfile when the idx file is not found on the
	// repository.
	ErrIdxNotFound = errors.New("idx file not found")
	// ErrMoreThanOnePackfile is returned by Packfile when more than one packfile
	// is found in the repository
	ErrMoreThanOnePackfile = errors.New("more than one packfile found")
	// ErrPackfileNotFound is returned by Packfile when the packfile is not found
	// on the repository.
	ErrPackfileNotFound = errors.New("packfile not found")
	// ErrMoreThanOneIdxfile is returned by Idxfile when more than one idxfile
	// is found in the repository
	ErrMoreThanOneIdxfile = errors.New("more than one idxfile found")
)

// The GitDir type represents a local git repository on disk. This
// type is not zero-value-safe, use the New function to initialize it.
type GitDir struct {
	fs   fs.FS
	path string
	refs map[string]core.Hash
}

// New returns a GitDir value ready to be used. The path argument must
// be the absolute path of a git repository directory (e.g.
// "/foo/bar/.git").
func New(fs fs.FS, path string) (*GitDir, error) {
	d := &GitDir{}
	d.fs = fs
	d.path = path

	if _, err := fs.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return d, nil
}

// Refs scans the git directory collecting references, which it returns.
// Symbolic references are resolved and included in the output.
func (d *GitDir) Refs() (map[string]core.Hash, error) {
	var err error

	d.refs = make(map[string]core.Hash)

	if err = d.addRefsFromPackedRefs(); err != nil {
		return nil, err
	}

	if err = d.addRefsFromRefDir(); err != nil {
		return nil, err
	}

	return d.refs, err
}

// Capabilities scans the git directory collection capabilities, which it returns.
func (d *GitDir) Capabilities() (*common.Capabilities, error) {
	c := common.NewCapabilities()

	err := d.addSymRefCapability(c)

	return c, err
}

func (d *GitDir) addSymRefCapability(cap *common.Capabilities) (err error) {
	f, err := d.fs.Open(d.fs.Join(d.path, "HEAD"))
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
	data := strings.TrimSpace(string(b))

	c := "symref"
	ref := strings.TrimPrefix(data, symRefPrefix)
	cap.Set(c, "HEAD:"+ref)

	return nil
}

// Packfile returns the path of the packfile in the repository.
func (d *GitDir) Packfile() (fs.FS, string, error) {
	p := d.pattern(true)

	list, err := d.fs.Glob(p)
	if err != nil {
		return nil, "", err
	}

	if len(list) == 0 {
		fmt.Println(p)
		return nil, "", ErrPackfileNotFound
	}

	if len(list) > 1 {
		return nil, "", ErrMoreThanOnePackfile
	}

	return d.fs, list[0], nil
}

func (d *GitDir) pattern(isPackfile bool) string {
	// packfile pattern: d.path + /objects/pack/pack-40hexs.pack
	//      idx pattern: d.path + /objects/pack/pack-40hexs.idx
	base := d.fs.Join(d.path, "objects")
	base = d.fs.Join(base, "pack")
	file := filePattern + extension(isPackfile)
	return d.fs.Join(base, file)
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
func (d *GitDir) Idxfile() (fs.FS, string, error) {
	p := d.pattern(false)

	list, err := d.fs.Glob(p)
	if err != nil {
		return nil, "", err
	}

	if len(list) == 0 {
		return nil, "", ErrIdxNotFound
	}

	if len(list) > 1 {
		return nil, "", ErrMoreThanOneIdxfile
	}

	return d.fs, list[0], nil
}
