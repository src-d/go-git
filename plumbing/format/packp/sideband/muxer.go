package sideband

import (
	"io"

	"gopkg.in/src-d/go-git.v4/plumbing/format/packp/pktline"
)

type Muxer struct {
	max  int
	e    *pktline.Encoder
	ch   chan *writeOp
	quit chan bool
}

const chLen = 1

func NewMuxer(t SidebandType, w io.Writer) *Muxer {
	max := MaxPackedSize64k
	if t == Sideband {
		max = MaxPackedSize
	}

	m := &Muxer{
		max:  max - chLen,
		e:    pktline.NewEncoder(w),
		ch:   make(chan *writeOp),
		quit: make(chan bool),
	}

	go m.doWrite()
	return m
}

func (m *Muxer) Write(p []byte) (int, error) {
	return m.WriteChannel(PackData, p)
}

func (m *Muxer) WriteChannel(t SidebandChannel, p []byte) (int, error) {
	wrote := 0
	size := len(p)
	for wrote < size {
		n, err := m.send(t, p[wrote:])
		wrote += n

		if err != nil {
			return wrote, err
		}
	}

	return wrote, nil
}

func (m *Muxer) send(t SidebandChannel, p []byte) (int, error) {
	sz := len(p)
	if sz > m.max {
		sz = m.max
	}

	op := newWriteOp(t, p[:sz])
	m.ch <- op
	return sz, <-op.err
}

func (m *Muxer) doWrite() {
	for {
		select {
		case <-m.quit:
			return
		case op := <-m.ch:
			op.err <- m.e.Encode(append(op.ch.Bytes(), op.b...))
		}
	}
}

func (m *Muxer) Close() error {
	m.quit <- true
	close(m.quit)
	close(m.ch)
	return nil
}

func newWriteOp(ch SidebandChannel, b []byte) *writeOp {
	return &writeOp{
		ch:  ch,
		b:   b,
		err: make(chan error),
	}
}

type writeOp struct {
	ch  SidebandChannel
	b   []byte
	err chan error
}
