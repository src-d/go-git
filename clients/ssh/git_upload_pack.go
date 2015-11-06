package ssh

import (
	"bufio"
	"bytes"
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

type GitUploadPackService struct {
	connected bool
	vcs       *vcsurl.RepoInfo
	client    *ssh.Client
}

func NewGitUploadPackService() *GitUploadPackService {
	return &GitUploadPackService{}
}

func (s *GitUploadPackService) Connect(ep common.Endpoint) (err error) {
	if s.connected {
		return fmt.Errorf("already connected")
	}

	s.vcs, err = vcsurl.Parse(string(ep))
	if err != nil {
		return fmt.Errorf("cannot parse vcs endpoint: %v", err)
	}

	url, err := vcsToUrl(s.vcs)
	if err != nil {
		return
	}

	s.client, err = connect(url.Host, url.User.Username())
	if err != nil {
		return fmt.Errorf("cannot connect: %v")
	}
	s.connected = true
	return
}

func vcsToUrl(vcs *vcsurl.RepoInfo) (u *url.URL, err error) {
	if vcs.VCS != vcsurl.Git {
		return nil, fmt.Errorf("only git repos are supported, found %s", vcs.VCS)
	}
	if vcs.RepoHost != vcsurl.GitHub {
		return nil, fmt.Errorf("only github.com host is supported, found %s", vcs.RepoHost)
	}
	s := "ssh://git@" + string(vcs.RepoHost) + ":22/" + vcs.FullName
	u, err = url.Parse(s)
	return
}

func connect(host, user string) (*ssh.Client, error) {

	// connect with ssh agent
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
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
		return nil, fmt.Errorf("failed to dial: %v", err)
	}

	return client, nil
}

func (s *GitUploadPackService) Info() (i *common.GitUploadPackInfo, err error) {
	if !s.connected {
		return nil, fmt.Errorf("not connected")
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("cannot open session: %v", err)
	}
	defer session.Close()

	out, err := session.Output("git-upload-pack " + s.vcs.FullName + ".git")
	if err != nil {
		return nil, fmt.Errorf("ssh.session.Output: %v", err)
	}

	i = common.NewGitUploadPackInfo()
	return i, i.Decode(pktline.NewDecoder(bytes.NewReader(out)))
}

func (s *GitUploadPackService) Disconnect() (err error) {
	if !s.connected {
		return fmt.Errorf("not connected")
	}
	if err = s.client.Close(); err != nil {
		return err
	}
	s.client = nil
	s.vcs = nil
	return nil
}

// TODO: fetch should really reuse the info session instead of openning a new
// one
func (s *GitUploadPackService) Fetch(r *common.GitUploadPackRequest) (io.ReadCloser, error) {
	if !s.connected {
		return nil, fmt.Errorf("not connected")
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("cannot open session: %v", err)
	}
	defer session.Close()

	si, err := session.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("cannot get ssh session stdin: %v", err)
	}

	so, err := session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("cannot get ssh session stdout: %v", err)
	}

	go func() {
		fmt.Fprintln(si, r.String())
		si.Close()
	}()

	err = session.Start("git-upload-pack " + s.vcs.FullName + ".git")
	if err != nil {
		return nil, fmt.Errorf("ssh.session.Start: %v", err)
	}
	session.Wait()

	// read until the header of the second answer
	soBuf := bufio.NewReader(so)
	token := "0000"
	for {
		line, err := soBuf.ReadString('\n')
		if err == io.EOF {
			return nil, fmt.Errorf("Bad answer format to git-upload-pack")
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
