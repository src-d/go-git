package gitdir

import (
	"bytes"
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

// Capabilities scans the git directory collection capabilities, which it returns.
func (d *Dir) Capabilities() (*common.Capabilities, error) {
	caps := common.NewCapabilities()

	err := d.addSymRefCapability(caps)

	return caps, err
}

func (d *Dir) addSymRefCapability(cap *common.Capabilities) (err error) {
	file, err := os.Open(filepath.Join(d.path, "HEAD"))
	if err != nil {
		return err
	}

	defer func() {
		errClose := file.Close()
		if err == nil {
			err = errClose
		}
	}()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	contents := strings.TrimSpace(string(bytes))

	capablity := "symref"
	ref := strings.TrimPrefix(contents, symRefPrefix)
	cap.Set(capablity, "HEAD:"+ref)

	return nil
}

// Packfile returns a readseeker of the packfile in the repository.
func (d *Dir) Packfile() (io.ReadSeeker, error) {
	pattern := d.pattern(true)
	list, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("packfile not found")
	}

	if len(list) > 1 {
		return nil, fmt.Errorf("found more than one packfile")
	}

	return os.Open(list[0])
}

func (d *Dir) pattern(isPackfile bool) (string, err) {
	// packfile pattern: dpath + /objects/pack/pack-40hexs.pack
	//      idx pattern: dpath + /objects/pack/pack-40hexs.idx
	var buf bytes.Buffer
	if _, err := buf.WriteString(d.path); err != nil {
		return "", nil
	}
	if _, err := buf.WriteByte(os.PathSeparator); err != nil {
		return "", nil
	}
	if _, err := buf.WriteString("objects"); err != nil {
		return "", nil
	}
	if _, err := buf.WriteByte(os.PathSeparator); err != nil {
		return "", nil
	}
	if _, err := buf.WriteString("pack"); err != nil {
		return "", nil
	}
	if _, err := buf.WriteByte(os.PathSeparator); err != nil {
		return "", nil
	}
	if _, err := buf.WriteString("pack-"); err != nil {
		return "", nil
	}
	for i := 0; i < 40; i++ {
		if _, err := buf.WriteString("[0-9a-f]"); err != nil {
			return "", nil
		}
	}
	if isPackfile {
		if _, err := buf.WriteString(".pack"); err != nil {
			return "", nil
		}
	} else {
		if _, err := buf.WriteString(".idx"); err != nil {
			return "", nil
		}
	}

	return buf.String()
}

// Idxfile returns a reader of the idx file in the repository.
// TODO: should it return a readcloser instead?
func (d *Dir) Idxfile() (io.Reader, error) {
	pattern := d.pattern(false)
	list, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, ErrIdxNotFound
	}

	if len(list) > 1 {
		return nil, fmt.Errorf("found more than one idxfile")
	}

	return os.Open(list[0])
}
