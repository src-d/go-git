// Package ssh implements a ssh client for go-git.
package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v4/plumbing/format/packp/advrefs"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packp/pktline"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packp/ulreq"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"

	"golang.org/x/crypto/ssh"
)

// New errors introduced by this package.
var (
	ErrInvalidAuthMethod      = errors.New("invalid ssh auth method")
	ErrAuthRequired           = errors.New("cannot connect: auth required")
	ErrNotConnected           = errors.New("not connected")
	ErrAlreadyConnected       = errors.New("already connected")
	ErrUploadPackAnswerFormat = errors.New("git-upload-pack bad answer format")
	ErrUnsupportedVCS         = errors.New("only git is supported")
	ErrUnsupportedRepo        = errors.New("only github.com is supported")
)

// Info returns the UploadPackInfo of the repository. The client must be
// connected with the repository (using the ConnectWithAuth() method) before
// using this method.
func (c *Client) FetchPackInfo() (*transport.UploadPackInfo, error) {
	if !c.connected {
		return nil, ErrNotConnected
	}

	session, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer func() {
		// the session can be closed by the other endpoint,
		// therefore we must ignore a close error.
		_ = session.Close()
	}()

	out, err := session.Output(c.getCommand())
	if err != nil {
		return nil, err
	}

	i := transport.NewUploadPackInfo()
	return i, i.Decode(bytes.NewReader(out))
}

// FetchPack returns a packfile for a given upload request.  It opens a new
// SSH session on a connected UploadPackService, sends the given
// upload request to the server and returns a reader for the received
// packfile.  Closing the returned reader will close the SSH session.
func (c *Client) FetchPack(req *transport.UploadPackRequest) (io.ReadCloser, error) {
	if !c.connected {
		return nil, ErrNotConnected
	}

	session, i, o, done, err := openSSHSession(c.client, c.getCommand())
	if err != nil {
		return nil, fmt.Errorf("cannot open SSH session: %s", err)
	}

	if err := fetchPack(i, o, req); err != nil {
		return nil, err
	}

	return &fetchSession{
		Reader:  o,
		session: session,
		done:    done,
	}, nil
}

type fetchSession struct {
	io.Reader
	session *ssh.Session
	done    <-chan error
}

// Close closes the session and collects the output state of the remote
// SSH command.
//
// If both the remote command and the closing of the session completes
// susccessfully it returns nil.
//
// If the remote command completes unsuccessfully or is interrupted by a
// signal, it returns the corresponding *ExitError.
//
// Otherwise, if clossing the SSH session fails it returns the close
// error.  Closing the session when the other has already close it is
// not cosidered an error.
func (f *fetchSession) Close() (err error) {
	if err := <-f.done; err != nil {
		return err
	}

	if err := f.session.Close(); err != nil && err != io.EOF {
		return err
	}

	return nil
}

func (c *Client) getCommand() string {
	directory := c.endpoint.Path
	directory = directory[1:]

	return fmt.Sprintf("git-upload-pack '%s'", directory)
}

var (
	nak = []byte("NAK")
	eol = []byte("\n")
)

// FetchPack implements the git-fetch-pack protocol.
//
// TODO support multi_ack mode
// TODO support multi_ack_detailed mode
// TODO support acks for common objects
// TODO build a proper state machine for all these processing options
func fetchPack(w io.WriteCloser, r io.Reader,
	req *transport.UploadPackRequest) error {

	fmt.Println("skipAdvRef")
	if err := skipAdvRef(r); err != nil {
		return fmt.Errorf("skipping advertised-refs: %s", err)
	}

	fmt.Println("sendUlReq")
	if err := sendUlReq(w, req); err != nil {
		return fmt.Errorf("sending upload-req message: %s", err)
	}

	if err := sendHaves(w, req); err != nil {
		return fmt.Errorf("sending haves message: %s", err)
	}

	if err := sendDone(w); err != nil {
		return fmt.Errorf("sending done message: %s", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("closing input: %s", err)
	}

	if err := readNAK(r); err != nil {
		return fmt.Errorf("reading NAK: %s", err)
	}

	return nil
}

func skipAdvRef(r io.Reader) error {
	d := advrefs.NewDecoder(r)
	ar := advrefs.New()

	return d.Decode(ar)
}

func sendUlReq(w io.Writer, req *transport.UploadPackRequest) error {
	ur := ulreq.New()
	ur.Wants = req.Wants
	ur.Depth = ulreq.DepthCommits(req.Depth)
	e := ulreq.NewEncoder(w)

	return e.Encode(ur)
}

func sendHaves(w io.Writer, req *transport.UploadPackRequest) error {
	e := pktline.NewEncoder(w)
	for _, have := range req.Haves {
		if err := e.Encodef("have %s\n", have); err != nil {
			return fmt.Errorf("sending haves for %q: %s", have, err)
		}
	}

	if len(req.Haves) != 0 {
		if err := e.Flush(); err != nil {
			return fmt.Errorf("sending flush-pkt after haves: %s", err)
		}
	}

	return nil
}

func sendDone(w io.Writer) error {
	e := pktline.NewEncoder(w)

	return e.Encodef("done\n")
}

func readNAK(r io.Reader) error {
	s := pktline.NewScanner(r)
	if !s.Scan() {
		return s.Err()
	}

	b := s.Bytes()
	b = bytes.TrimSuffix(b, eol)
	if !bytes.Equal(b, nak) {
		return fmt.Errorf("expecting NAK, found %q instead", string(b))
	}

	return nil
}
