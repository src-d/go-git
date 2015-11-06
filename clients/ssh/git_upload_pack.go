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

var (
	ErrInvalidAuthMethod      = errors.New("invalid ssh auth method: a ssh.SSHAuthMethod should be provided.")
	ErrAuthRequired           = errors.New("cannot connect: auth required.")
	ErrNotConnected           = errors.New("not connected")
	ErrAlreadyConnected       = errors.New("already connected")
	ErrUploadPackAnswerFormat = errors.New("git-upload-pack bad answer format")
	ErrUnsupportedVCS         = errors.New("only git is supported")
	ErrUnsupportedRepo        = errors.New("only github.com is supported")
)

type SSHAuthMethod interface {
	common.AuthMethod
}

type SSHAgent struct {
	env string
}

// Env is the ssh agent Unix socket env var.
// an empty env wil set the default "SSH_AUTH_SOCK"
func NewSSHAgent(env string) *SSHAgent {
	if env == "" {
		return &SSHAgent{"SSH_AUTH_SOCK"}
	}
	return &SSHAgent{env}
}

func (a *SSHAgent) Name() string {
	return "ssh agent"
}

func (a *SSHAgent) String() string {
	return a.Name()
}

type GitUploadPackService struct {
	connected bool
	vcs       *vcsurl.RepoInfo
	client    *ssh.Client
	auth      SSHAuthMethod
}

func NewGitUploadPackService() *GitUploadPackService {
	return &GitUploadPackService{}
}

func (s *GitUploadPackService) Connect(ep common.Endpoint) (err error) {
	return ErrAuthRequired
}

func (s *GitUploadPackService) ConnectWithAuth(ep common.Endpoint, auth common.AuthMethod) (err error) {
	if s.connected {
		return ErrAlreadyConnected
	}

	sshAuth, ok := auth.(SSHAuthMethod)
	if !ok {
		return ErrInvalidAuthMethod
	}
	s.auth = sshAuth

	s.vcs, err = vcsurl.Parse(string(ep))
	if err != nil {
		return err
	}

	url, err := vcsToUrl(s.vcs)
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

func vcsToUrl(vcs *vcsurl.RepoInfo) (u *url.URL, err error) {
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

func connect(host, user string, auth SSHAuthMethod) (*ssh.Client, error) {

	agentAuth, ok := auth.(*SSHAgent)
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

func (s *GitUploadPackService) Disconnect() (err error) {
	if !s.connected {
		return ErrNotConnected
	}
	s.connected = false
	return s.client.Close()
}

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
		line, err := soBuf.ReadString('\n')
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
