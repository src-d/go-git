package gitdir

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v3/core"
)

var (
	ErrPackedRefsDuplicatedRef = errors.New("duplicated ref found in packed-ref file")
	ErrPackedRefsBadFormat     = errors.New("malformed packed-ref")
	ErrSymRefTargetNotFound    = errors.New("symbolic reference target not found")
)

const (
	symRefPrefix = "ref: "
)

func (d *Dir) initRefsFromPackedRefs() (m map[string]core.Hash, err error) {
	result := make(map[string]core.Hash)

	path := filepath.Join(d.path, packedRefsPath)
	file, err := os.Open(path)
	if err != nil {
		if err == os.ErrNotExist {
			return result, nil
		}
		return nil, err
	}
	defer func() {
		errClose := file.Close()
		if err == nil {
			err = errClose
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if err = processLine(line, result); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// process lines from a packed-refs file
func processLine(line string, refs map[string]core.Hash) error {
	switch line[0] {
	case '#': // comment - ignore
		return nil
	case '^': // annotated tag commit of the previous line - ignore
		return nil
	default:
		words := strings.Split(line, " ") // hash then ref
		if len(words) != 2 {
			return ErrPackedRefsBadFormat
		}
		hash, ref := words[0], words[1]

		if _, ok := refs[ref]; ok {
			return ErrPackedRefsDuplicatedRef
		}
		refs[ref] = core.NewHash(hash)
	}

	return nil
}

func (d *Dir) addRefsFromRefDir() error {
	return d.walkTree("refs")
}

func (d *Dir) walkTree(relPath string) error {
	files, err := ioutil.ReadDir(filepath.Join(d.path, relPath))
	if err != nil {
		return err
	}

	for _, file := range files {
		newRelPath := filepath.Join(relPath, file.Name())

		if file.IsDir() {
			if err = d.walkTree(newRelPath); err != nil {
				return err
			}
		} else {
			filePath := filepath.Join(d.path, newRelPath)
			hash, err := d.readHashFile(filePath)
			if err != nil {
				return err
			}
			d.refs[newRelPath] = hash
		}
	}

	return nil
}

// ReadHashFile reads a single hash from a file.  If a symbolic
// reference is found instead of a hash, the reference is resolved and
// the proper hash is returned.
func (d *Dir) readHashFile(path string) (core.Hash, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return core.ZeroHash, err
	}
	content := strings.TrimSpace(string(bytes))

	if isSymRef(content) {
		return d.resolveSymRef(content)
	}

	return core.NewHash(content), nil
}

func isSymRef(contents string) bool {
	return strings.HasPrefix(contents, symRefPrefix)
}

func (d *Dir) resolveSymRef(symRef string) (core.Hash, error) {
	ref := strings.TrimPrefix(symRef, symRefPrefix)

	hash, ok := d.refs[ref]
	if !ok {
		return core.ZeroHash, ErrSymRefTargetNotFound
	}

	return hash, nil
}
