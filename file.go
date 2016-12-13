package git

import (
	"bytes"
	"io"
	"os"
	"strings"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

// File represents git file objects.
type File struct {
	Name string
	Mode os.FileMode
	Blob
}

// NewFile returns a File based on the given blob object
func NewFile(name string, m os.FileMode, b *Blob) *File {
	return &File{Name: name, Mode: m, Blob: *b}
}

// Contents returns the contents of a file as a string.
func (f *File) Contents() (content string, err error) {
	reader, err := f.Reader()
	if err != nil {
		return "", err
	}
	defer checkClose(reader, &err)

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(reader); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Lines returns a slice of lines from the contents of a file, stripping
// all end of line characters. If the last line is empty (does not end
// in an end of line), it is also stripped.
func (f *File) Lines() ([]string, error) {
	content, err := f.Contents()
	if err != nil {
		return nil, err
	}

	splits := strings.Split(content, "\n")
	// remove the last line if it is empty
	if splits[len(splits)-1] == "" {
		return splits[:len(splits)-1], nil
	}

	return splits, nil
}

type FileIter struct {
	s storer.EncodedObjectStorer
	w TreeWalker
}

func NewFileIter(s storer.EncodedObjectStorer, t *Tree) *FileIter {
	return &FileIter{s: s, w: *NewTreeWalker(s, t, true)}
}

func (iter *FileIter) Next() (*File, error) {
	for {
		name, entry, err := iter.w.Next()
		if err != nil {
			return nil, err
		}

		if entry.Mode.IsDir() {
			continue
		}

		o, err := iter.s.EncodedObject(plumbing.BlobObject, entry.Hash)
		if err != nil {
			return nil, err
		}

		blob, err := DecodeBlob(o)
		if err != nil {
			return nil, err
		}

		return NewFile(name, entry.Mode, blob), nil
	}
}

// ForEach call the cb function for each file contained on this iter until
// an error happends or the end of the iter is reached. If plumbing.ErrStop is sent
// the iteration is stop but no error is returned. The iterator is closed.
func (iter *FileIter) ForEach(cb func(*File) error) error {
	defer iter.Close()

	for {
		f, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		if err := cb(f); err != nil {
			if err == storer.ErrStop {
				return nil
			}

			return err
		}
	}
}

func (iter *FileIter) Close() {
	iter.w.Close()
}
