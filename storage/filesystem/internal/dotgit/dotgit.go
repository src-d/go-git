package dotgit

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/src-d/go-git.v4/core"
	"gopkg.in/src-d/go-git.v4/utils/fs"
)

const (
	suffix         = ".git"
	packedRefsPath = "packed-refs"
	configPath     = "config"
)

var (
	// ErrNotFound is returned by New when the path is not found.
	ErrNotFound = errors.New("path not found")
	// ErrIdxNotFound is returned by Idxfile when the idx file is not found
	ErrIdxNotFound = errors.New("idx file not found")
	// ErrPackfileNotFound is returned by Packfile when the packfile is not found
	ErrPackfileNotFound = errors.New("packfile not found")
	// ErrConfigNotFound is returned by Config when the config is not found
	ErrConfigNotFound = errors.New("config file not found")
)

// The DotGit type represents a local git repository on disk. This
// type is not zero-value-safe, use the New function to initialize it.
type DotGit struct {
	fs   fs.FS
	path string
}

// New returns a DotGit value ready to be used. The path argument must
// be the absolute path of a git repository directory (e.g.
// "/foo/bar/.git").
func New(fs fs.FS, path string) (*DotGit, error) {
	d := &DotGit{fs: fs, path: path}
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
func (d *DotGit) Refs() ([]*core.Reference, error) {
	var refs []*core.Reference
	_ = d.addRefsFromPackedRefs(&refs)

	if err := d.addRefsFromRefDir(&refs); err != nil {
		return nil, err
	}

	if err := d.addRefFromHEAD(&refs); err != nil {
		return nil, err
	}

	return refs, nil
}

// Packfile returns the path of the packfile (really, it returns the
// path of the first file in the "objects/pack/" directory with a
// ".pack" extension.
func (d *DotGit) Packfile() (fs.FS, string, error) {
	packDir := d.fs.Join(d.path, "objects", "pack")
	files, err := d.fs.ReadDir(packDir)
	if err != nil {
		return nil, "", err
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".pack") {
			return d.fs, d.fs.Join(packDir, f.Name()), nil
		}
	}

	return nil, "", ErrPackfileNotFound
}

// Idxfile returns the path of the idx file (really, it returns the
// path of the first file in the "objects/pack/" directory with an
// ".idx" extension.
func (d *DotGit) Idxfile() (fs.FS, string, error) {
	packDir := d.fs.Join(d.path, "objects", "pack")
	files, err := d.fs.ReadDir(packDir)
	if err != nil {
		return nil, "", err
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".idx") {
			return d.fs, d.fs.Join(packDir, f.Name()), nil
		}
	}

	return nil, "", ErrIdxNotFound
}

// Config returns the path of the config file
func (d *DotGit) Config() (fs.FS, string, error) {
	configFile := d.fs.Join(d.path, configPath)
	if _, err := d.fs.Stat(configFile); err != nil {
		if os.IsNotExist(err) {
			return nil, "", ErrNotFound
		}

		return nil, "", err
	}

	return d.fs, configFile, nil
}

func (d *DotGit) Objectfiles() (fs.FS, []core.Hash, error) {
	dotGitobjcts := d.fs.Join(d.path, "objects")
	files, err := d.fs.ReadDir(dotGitobjcts)
	if err != nil {
		return nil, nil, err
	}

	var objDirs []string
	reDir, _ := regexp.Compile("[a-z0-9]{2}")
	for _, f := range files {
		if f.IsDir() && reDir.MatchString(f.Name()) {
			objDirs = append(objDirs, f.Name())
		}
	}

	var objects []core.Hash
	reObj, _ := regexp.Compile("[a-z0-9]{38}")
	for _, dir := range objDirs {
		objs, err := d.fs.ReadDir(d.fs.Join(dotGitobjcts, dir))

		if err != nil {
			return nil, nil, err
		}

		for _, obj := range objs {
			if reObj.MatchString(obj.Name()) {
				name := dir + obj.Name()
				fmt.Println(name)
				objects = append(objects, core.NewHash(name))
			}
		}
	}

	return d.fs, objects, nil
}

func (d *DotGit) Objectfile(h core.Hash) (fs.FS, string, error) {
	hash := h.String()
	objFile := d.fs.Join(d.path, "objects", hash[0:2], hash[2:40])

	if _, err := d.fs.Stat(objFile); err != nil {
		if os.IsNotExist(err) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	return d.fs, objFile, nil
}
