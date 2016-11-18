package sideband

import (
	"errors"
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v4/plumbing/format/packp/pktline"
)

var ErrMaxPackedExceeded = errors.New("max. packed size exceeded")

type SidebandType int8

const (
	Sideband    SidebandType = iota
	Sideband64k SidebandType = iota

	MaxPackedSize    = 1000
	MaxPackedSize64k = 65520
)

type SidebandChannel byte

const (
	PackData          SidebandChannel = 1
	ProgressMessage   SidebandChannel = 2
	FatalErrorMessage SidebandChannel = 3
)

type Demuxer struct {
	t SidebandType
	r io.ReadCloser
	s *pktline.Scanner

	max     int
	pending []byte
}

func NewDemuxer(t SidebandType, r io.ReadCloser) io.ReadCloser {
	max := MaxPackedSize64k
	if t == Sideband {
		max = MaxPackedSize
	}

	return &Demuxer{
		t:   t,
		r:   r,
		max: max,
		s:   pktline.NewScanner(r),
	}
}

func (d *Demuxer) Read(b []byte) (int, error) {
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

	switch SidebandChannel(content[0]) {
	case PackData:
		return content[1:], err
	default:
		fmt.Printf("remote: %s", content[1:])
		return nil, err
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

func (d *Demuxer) Close() error {
	return d.r.Close()
}
