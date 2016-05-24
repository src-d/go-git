package gitdir

import (
	"errors"
	"fmt"
	"io/ioutil"
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
}

// New returns a Dir value ready to be used. The path argument must be
// an existing git repository directory (e.g. "foo/bar/.git") on which
// the "git gc" command has been run.
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

// Returns the references in a git directory.
func (d *Dir) Refs() (map[string]core.Hash, error) {
	refs, err := d.refsFromPackedRefs()
	if err != nil {
		return nil, err
	}

	if err := d.refsFromRefDir(refs); err != nil {
		return nil, err
	}

	return refs, err
}

func (d *Dir) refsFromRefDir(result map[string]core.Hash) error {
	return nil
}

/*
func refsTree(basePath, relPath string, result map[string]core.Hash) error {
	fmt.Printf("calling refs(%s, %s, %v)\n", basePath, relPath, result)
	files, err := ioutil.ReadDir(basePath + relPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		newRelPath := relPath + file.Name()
		if file.IsDir() {
			refs(basePath, newRelPath+"/", result)
		} else {
			_ = basePath + newRelPath
				hash, err := ReadHashFile(path)
				if err != nil {
					return err
				}
				result[newRelPath] = core.NewHash(string(content))
		}
	}

	return nil
}
*/

// ReadHashFile reads a single hash from a file.  If a symbolic
// reference is found instead of a hash, the reference is resolved and
// the proper hash is returned.
func ReadHashFile(repo, relPath string) (core.Hash, error) {
	content, err := ioutil.ReadFile(repo)
	if err != nil {
		return core.ZeroHash, err
	}
	fmt.Println(string(content))
	return core.ZeroHash, nil
}
