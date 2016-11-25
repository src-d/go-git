package common

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"gopkg.in/src-d/go-git.v4/plumbing/format/pktline"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

type Command interface {
	SetAuth(transport.AuthMethod) error
	Start() error
	StderrPipe() (io.Reader, error)
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.Reader, error)
	Wait() error
	io.Closer
}

type Commander interface {
	Command(cmd string, ep transport.Endpoint) (Command, error)
}

type client struct {
	cmdr Commander
}

// NewClient creates a new client using a CommandRunner.
func NewClient(runner Commander) transport.Client {
	return &client{runner}
}

func (c *client) NewFetchPackSession(ep transport.Endpoint) (
	transport.FetchPackSession, error) {

	return c.newSession(transport.UploadPackServiceName, ep)
}

func (c *client) NewSendPackSession(ep transport.Endpoint) (
	transport.SendPackSession, error) {

	return nil, errors.New("git send-pack not supported")
}

type session struct {
	Stdin   io.WriteCloser
	Stdout  io.Reader
	Stderr  io.Reader
	Command Command

	advRefsRun bool
}

func (c *client) newSession(s string, ep transport.Endpoint) (*session, error) {
	cmd, err := c.cmdr.Command(s, ep)
	if err != nil {
		return nil, err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &session{
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Command: cmd,
	}, nil
}

func (s *session) SetAuth(auth transport.AuthMethod) error {
	return s.Command.SetAuth(auth)
}

func (s *session) AdvertisedReferences() (*packp.AdvRefs, error) {
	if s.advRefsRun {
		return nil, transport.ErrAdvertistedReferencesAlreadyCalled
	}

	defer func() { s.advRefsRun = true }()

	ar := packp.NewAdvRefs()
	if err := ar.Decode(s.Stdout); err != nil {
		if err != packp.ErrEmptyAdvRefs {
			return nil, err
		}

		_ = s.Stdin.Close()

		scan := bufio.NewScanner(s.Stderr)
		if !scan.Scan() {
			return nil, transport.ErrEmptyRemoteRepository
		}

		if isRepoNotFoundError(string(scan.Bytes())) {
			return nil, transport.ErrRepositoryNotFound
		}

		return nil, err
	}

	return ar, nil
}

// FetchPack returns a packfile for a given upload request.
// Closing the returned reader will close the SSH session.
func (s *session) FetchPack(req *packp.UploadPackRequest) (io.ReadCloser, error) {
	if req.IsEmpty() {
		return nil, transport.ErrEmptyUploadPackRequest
	}

	if !s.advRefsRun {
		if _, err := s.AdvertisedReferences(); err != nil {
			return nil, err
		}
	}

	if err := fetchPack(s.Stdin, s.Stdout, req); err != nil {
		return nil, err
	}

	r, err := ioutil.NonEmptyReader(s.Stdout)
	if err == ioutil.ErrEmptyReader {
		if c, ok := s.Stdout.(io.Closer); ok {
			_ = c.Close()
		}

		return nil, transport.ErrEmptyUploadPackRequest
	}

	if err != nil {
		return nil, err
	}

	wc := &waitCloser{s.Command}
	rc := ioutil.NewReadCloser(r, wc)

	return rc, nil
}

func (s *session) Close() error {
	return s.Command.Close()
}

const (
	githubRepoNotFoundErr    = "ERROR: Repository not found."
	bitbucketRepoNotFoundErr = "conq: repository does not exist."
)

func isRepoNotFoundError(s string) bool {
	if strings.HasPrefix(s, githubRepoNotFoundErr) {
		return true
	}

	if strings.HasPrefix(s, bitbucketRepoNotFoundErr) {
		return true
	}

	return false
}

var (
	nak = []byte("NAK")
	eol = []byte("\n")
)

// fetchPack implements the git-fetch-pack protocol.
//
// TODO support multi_ack mode
// TODO support multi_ack_detailed mode
// TODO support acks for common objects
// TODO build a proper state machine for all these processing options
func fetchPack(w io.WriteCloser, r io.Reader,
	req *packp.UploadPackRequest) error {

	if err := req.UploadRequest.Encode(w); err != nil {
		return fmt.Errorf("sending upload-req message: %s", err)
	}

	if err := req.UploadHaves.Encode(w); err != nil {
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

type waitCloser struct {
	Command Command
}

// Close waits until the command exits and returns error, if any.
func (c *waitCloser) Close() error {
	return c.Command.Wait()
}
