package readcounter

import (
	"fmt"
	"io"
)

// ReadCounter is a reader that keeps track of the number of bytes readed.
type ReadCounter struct {
	io.Reader
	count int64
}

// New returns a new ReadCounter for the given stream r.
func New(r io.Reader) *ReadCounter {
	return &ReadCounter{Reader: r, count: 0}
}

func (t *ReadCounter) Read(p []byte) (n int, err error) {
	n, err = t.Reader.Read(p)
	if err != nil {
		return 0, err
	}

	t.count += int64(n)

	return n, err
}

// ReadByte reads a byte from the readcounter.
func (t *ReadCounter) ReadByte() (c byte, err error) {
	var p [1]byte
	n, err := t.Reader.Read(p[:])
	if err != nil {
		return 0, err
	}

	if n > 1 {
		return 0, fmt.Errorf("read %d bytes, should have read just 1", n)
	}

	t.count++

	return p[0], nil
}

// Count returns the number of bytes read so far.
func (t *ReadCounter) Count() int64 {
	return t.count
}
