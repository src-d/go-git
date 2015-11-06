// Package ssh implements a ssh client for go-git.
//
// The Connect() method is not allowed, use ConnectWithAuth() instead.
package ssh

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"

	"gopkg.in/src-d/go-git.v2/clients/common"
	"gopkg.in/src-d/go-git.v2/formats/pktline"

	"github.com/sourcegraph/go-vcsurl"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// New errors introduced by this package.
var (
	ErrInvalidAuthMethod      = errors.New("invalid ssh auth method: a ssh.SSHAuthMethod should be provided")
	ErrAuthRequired           = errors.New("cannot connect: auth required")
	ErrNotConnected           = errors.New("not connected")
	ErrAlreadyConnected       = errors.New("already connected")
	ErrUploadPackAnswerFormat = errors.New("git-upload-pack bad answer format")
	ErrUnsupportedVCS         = errors.New("only git is supported")
	ErrUnsupportedRepo        = errors.New("only github.com is supported")
)

// AuthMethod is the interface all auth methods for the ssh client
// must implement.
type AuthMethod interface {
	common.AuthMethod
}

// Agent is an authentication method that uses a running ssh agent to
// handle all ssh authentication. Its zero value is not safe, use
// NewSSHAgent() instead.
type Agent struct {
	env string
}

// NewSSHAgent initialises a SSHAgent. Env is the ssh agent Unix socket
// environment variable. Pass an empty string as env to use the default
// value "SSH_AUTH_SOCK".
func NewSSHAgent(env string) *Agent {
	if env == "" {
		return &Agent{"SSH_AUTH_SOCK"}
	}
	return &Agent{env}
}

// Name returns an id for this authentication method.
func (a *Agent) Name() string {
	return "ssh agent"
}

func (a *Agent) String() string {
	return a.Name()
}

// GitUploadPackService for SSH clients.
// The zero value is safe to use.
// TODO: remove NewGitUploadPackService().
type GitUploadPackService struct {
	connected bool
	vcs       *vcsurl.RepoInfo
	client    *ssh.Client
	auth      AuthMethod
}

// NewGitUploadPackService initialises a GitUploadPackService.
// TODO: remove this, as the struct is zero-value safe.
func NewGitUploadPackService() *GitUploadPackService {
	return &GitUploadPackService{}
}

// Connect cannot be used with SSH clients and always return an error. Use ConnectWithAuth instead.
func (s *GitUploadPackService) Connect(ep common.Endpoint) (err error) {
	return ErrAuthRequired
}

// ConnectWithAuth connects to ep using SSH. Authentication is handled by auth.
func (s *GitUploadPackService) ConnectWithAuth(ep common.Endpoint, auth common.AuthMethod) (err error) {
	if s.connected {
		return ErrAlreadyConnected
	}

	sshAuth, ok := auth.(AuthMethod)
	if !ok {
		return ErrInvalidAuthMethod
	}
	s.auth = sshAuth

	s.vcs, err = vcsurl.Parse(string(ep))
	if err != nil {
		return err
	}

	url, err := vcsToURL(s.vcs)
	if err != nil {
		return
	}

	s.client, err = connect(url.Host, url.User.Username(), sshAuth)
	if err != nil {
		return err
	}
	s.connected = true
	return
}

func vcsToURL(vcs *vcsurl.RepoInfo) (u *url.URL, err error) {
	if vcs.VCS != vcsurl.Git {
		return nil, ErrUnsupportedVCS
	}
	if vcs.RepoHost != vcsurl.GitHub {
		return nil, ErrUnsupportedRepo
	}
	s := "ssh://git@" + string(vcs.RepoHost) + ":22/" + vcs.FullName
	u, err = url.Parse(s)
	return
}

func connect(host, user string, auth AuthMethod) (*ssh.Client, error) {

	agentAuth, ok := auth.(*Agent)
	if !ok {
		return nil, ErrInvalidAuthMethod
	}

	// connect with ssh agent
	conn, err := net.Dial("unix", os.Getenv(agentAuth.env))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	agent := agent.NewClient(conn)
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeysCallback(agent.Signers)},
	}

	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Info returns the GitUploadPackInfo of the repository.
// The client must be connected with the repository (using
// the ConnectWithAuth() method) before using this
// method.
func (s *GitUploadPackService) Info() (i *common.GitUploadPackInfo, err error) {
	if !s.connected {
		return nil, ErrNotConnected
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	out, err := session.Output("git-upload-pack " + s.vcs.FullName + ".git")
	if err != nil {
		return nil, err
	}

	i = common.NewGitUploadPackInfo()
	return i, i.Decode(pktline.NewDecoder(bytes.NewReader(out)))
}

// Disconnect the SSH client.
func (s *GitUploadPackService) Disconnect() (err error) {
	if !s.connected {
		return ErrNotConnected
	}
	s.connected = false
	return s.client.Close()
}

// Fetch retrieves the GitUploadPack form the repository.
// You must be connected to the repository before using this method
// (using the ConnectWithAuth() method).
// TODO: fetch should really reuse the info session instead of openning a new
// one
func (s *GitUploadPackService) Fetch(r *common.GitUploadPackRequest) (io.ReadCloser, error) {
	if !s.connected {
		return nil, ErrNotConnected
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	si, err := session.StdinPipe()
	if err != nil {
		return nil, err
	}

	so, err := session.StdoutPipe()
	if err != nil {
		return nil, err
	}

	go func() {
		fmt.Fprintln(si, r.String())
		si.Close()
	}()

	err = session.Start("git-upload-pack " + s.vcs.FullName + ".git")
	if err != nil {
		return nil, err
	}
	session.Wait()

	// read until the header of the second answer
	soBuf := bufio.NewReader(so)
	token := "0000"
	for {
		var line string
		line, err = soBuf.ReadString('\n')
		if err == io.EOF {
			return nil, ErrUploadPackAnswerFormat
		}
		if line[0:len(token)] == token {
			break
		}
	}

	data, err := ioutil.ReadAll(soBuf)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(data)
	return ioutil.NopCloser(buf), nil
}
