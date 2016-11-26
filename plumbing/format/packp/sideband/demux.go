package sideband

import (
	"errors"
	"fmt"
	"io"

	"bytes"

	"gopkg.in/src-d/go-git.v4/plumbing/format/pktline"
)

// ErrMaxPackedExceeded returned by Read, if the maximum packed size is exceeded
var ErrMaxPackedExceeded = errors.New("max. packed size exceeded")

// Progress allows to read the progress information
type Progress interface {
	io.Reader
}

// Demuxer demultiplex the progress reports and error info interleaved with the
// packfile itself.
type Demuxer struct {
	t Type
	r io.Reader
	s *pktline.Scanner

	max     int
	pending []byte

	// Progress contains progress information
	Progress Progress
}

// NewDemuxer returns a new Demuxer for the given t and read from r
func NewDemuxer(t Type, r io.Reader) *Demuxer {
	max := MaxPackedSize64k
	if t == Sideband {
		max = MaxPackedSize
	}

	return &Demuxer{
		t:        t,
		r:        r,
		max:      max,
		s:        pktline.NewScanner(r),
		Progress: bytes.NewBuffer(nil),
	}
}

// Read reads up to len(p) bytes from the PackData channel into p, an error can
// be return if an error happends when reading or if a message is sent in the
// ErrorMessage channel.
//
// If a ProgressMessage is read, it's not copied into b, intead of this is
// is stored, can be read through the reader Progress, the n value returned is
// zero, err is nil unless an error reading happends.
func (d *Demuxer) Read(b []byte) (n int, err error) {
	var read, req int

	req = len(b)
	for read < req {
		n, err := d.doRead(b[read:req])
		read += n

		if err == io.EOF {
			break
		}

		if err != nil {
			return read, err
		}
	}

	return read, nil
}

func (d *Demuxer) doRead(b []byte) (int, error) {
	read, err := d.nextPackData()
	size := len(read)
	wanted := len(b)

	if size > wanted {
		d.pending = read[wanted:size]
	}

	if wanted > size {
		wanted = size
	}

	size = copy(b, read[:wanted])
	return size, err
}

func (d *Demuxer) nextPackData() ([]byte, error) {
	content := d.readPending()
	if len(content) != 0 {
		return content, nil
	}

	if !d.s.Scan() {
		return nil, io.EOF
	}

	content = d.s.Bytes()
	err := d.s.Err()

	size := len(content)
	if size == 0 {
		return nil, err
	} else if size > d.max {
		return nil, ErrMaxPackedExceeded
	}

	switch Channel(content[0]) {
	case PackData:
		return content[1:], err
	case ProgressMessage:
		_, err := d.Progress.(io.Writer).Write(content[1:])
		return nil, err
	case ErrorMessage:
		return nil, fmt.Errorf("unexepcted error: %s", content[1:])
	default:
		return nil, fmt.Errorf("unknown channel %s", content)
	}
}

func (d *Demuxer) readPending() (b []byte) {
	if len(d.pending) == 0 {
		return nil
	}

	content := d.pending
	d.pending = make([]byte, 0)

	return content
}
