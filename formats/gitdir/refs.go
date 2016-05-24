package gitdir

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v3/core"
)

var (
	ErrPackedRefsDuplicatedRef = errors.New("duplicated ref found in packed-ref file")
	ErrPackedRefsBadFormat     = errors.New("malformed packed-ref")
)

func (d *Dir) refsFromPackedRefs() (m map[string]core.Hash, err error) {
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
