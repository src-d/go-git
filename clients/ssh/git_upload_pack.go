package ssh

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"

	"github.com/sourcegraph/go-vcsurl"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"gopkg.in/src-d/go-git.v2/clients/common"
	"gopkg.in/src-d/go-git.v2/formats/pktline"
)

type GitUploadPackService struct {
	vcs     *vcsurl.RepoInfo
	client  *ssh.Client
	session *ssh.Session
	user    string
}

func NewGitUploadPackService() *GitUploadPackService {
	return &GitUploadPackService{}
}

func (s *GitUploadPackService) Connect(ep common.Endpoint) (err error) {
	s.vcs, err = vcsurl.Parse(string(ep))
	if err != nil {
		return
	}

	url, err := vcsToUrl(s.vcs)
	if err != nil {
		return
	}

	s.client, s.session, err = connect(url.Host, url.User.Username())
	if err != nil {
		return
	}

	return
}

func vcsToUrl(vcs *vcsurl.RepoInfo) (u *url.URL, err error) {
	if vcs.VCS != vcsurl.Git {
		return nil, fmt.Errorf("only git repos are supported, found %s\n", vcs.VCS)
	}
	if vcs.RepoHost != vcsurl.GitHub {
		return nil, fmt.Errorf("only github.com host is supported, found %s\n", vcs.RepoHost)
	}
	s := "ssh://git@" + string(vcs.RepoHost) + ":22/" + vcs.FullName
	u, err = url.Parse(s)
	return
}

func connect(host, user string) (*ssh.Client, *ssh.Session, error) {

	// try ssh-agent first
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return connectWithPasswd(host, user)
	}
	defer conn.Close()

	agent := agent.NewClient(conn)
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeysCallback(agent.Signers)},
	}

	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to dial:", err)
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create session:", err)
	}

	return client, session, nil
}

func connectWithPasswd(host, user string) (*ssh.Client, *ssh.Session, error) {
	var pass string
	fmt.Print("Password: ")
	fmt.Scanf("%s\n", &pass)

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.Password(pass)},
	}

	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, session, nil
}

func (s *GitUploadPackService) Info() (*common.GitUploadPackInfo, error) {
	out, err := s.session.CombinedOutput("git-upload-pack " + s.vcs.FullName + ".git")
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(out)
	dec := pktline.NewDecoder(buf)
	return common.NewGitUploadPackInfo(dec)
}

func (s *GitUploadPackService) Disconnect() (err error) {
	if s.client == nil {
		return fmt.Errorf("cannot close a non-connected ssh upload pack service")
	}
	if err = s.client.Close(); err != nil {
		return err
	}
	s.client = nil
	s.session = nil
	s.vcs = nil
	return nil
}

func (s *GitUploadPackService) Fetch(r *common.GitUploadPackRequest) (io.ReadCloser, error) {
	return nil, nil
}
