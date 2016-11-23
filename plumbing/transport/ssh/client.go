package ssh

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	connected bool
	endpoint  transport.Endpoint
	client    *ssh.Client
	auth      AuthMethod
}

func NewClient(ep transport.Endpoint) transport.Client {
	return &Client{endpoint: ep}
}

// Connect connects to the SSH server, unless a AuthMethod was set with SetAuth
// method, by default uses an auth method based on PublicKeysCallback, it
// connects to a SSH agent, using the address stored in the SSH_AUTH_SOCK
// environment var
func (c *Client) Connect() error {
	if c.connected {
		return ErrAlreadyConnected
	}

	if err := c.setAuthFromEndpoint(); err != nil {
		return err
	}

	var err error
	c.client, err = ssh.Dial("tcp", c.getHostWithPort(), c.auth.clientConfig())
	if err != nil {
		return err
	}

	c.connected = true
	return nil
}

// Disconnect the SSH client.
func (s *Client) Disconnect() error {
	if !s.connected {
		return ErrNotConnected
	}
	s.connected = false
	return s.client.Close()
}

func (c *Client) getHostWithPort() string {
	host := c.endpoint.Host
	if strings.Index(c.endpoint.Host, ":") == -1 {
		host += ":22"
	}

	return host
}

func (c *Client) setAuthFromEndpoint() error {
	var u string
	if info := c.endpoint.User; info != nil {
		u = info.Username()
	}

	var err error
	c.auth, err = NewSSHAgentAuth(u)
	return err
}

// SetAuth sets the AuthMethod
func (c *Client) SetAuth(auth transport.AuthMethod) error {
	var ok bool
	c.auth, ok = auth.(AuthMethod)
	if !ok {
		return ErrInvalidAuthMethod
	}

	return nil
}

func openSSHSession(c *ssh.Client, cmd string) (
	*ssh.Session, io.WriteCloser, io.Reader, <-chan error, error) {

	session, err := c.NewSession()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("cannot open SSH session: %s", err)
	}

	i, err := session.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("cannot pipe remote stdin: %s", err)
	}

	o, err := session.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("cannot pipe remote stdout: %s", err)
	}

	done := make(chan error)
	go func() {
		done <- session.Run(cmd)
	}()

	return session, i, o, done, nil
}
