package gitdir

import (
	"errors"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v3/core"
)

const (
	suffix         = ".git"
	refsDir        = "refs/"
	packedRefsPath = "packed-refs"
)

var (
	ErrBadGitDirName = errors.New(`Bad git dir name (must end in ".git")`)
)

type Dir struct {
	path string
	refs map[string]core.Hash
}

// New returns a Dir value ready to be used. The path argument must be
// an existing git repository directory (e.g. "/foo/bar/.git").
func New(path string) (*Dir, error) {
	dir := &Dir{}
	var err error

	dir.path, err = cleanPath(path)
	if err != nil {
		return nil, err
	}

	if dir.isInvalidPath() {
		return nil, ErrBadGitDirName
	}

	return dir, nil
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

	d.refs, err = d.initRefsFromPackedRefs()
	if err != nil {
		return nil, err
	}

	if err = d.addRefsFromRefDir(); err != nil {
		return nil, err
	}

	return d.refs, err
}
