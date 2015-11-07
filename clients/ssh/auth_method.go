package ssh

import (
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"gopkg.in/src-d/go-git.v2/clients/common"
)

// AuthMethod is the interface all auth methods for the ssh client
// must implement.
type AuthMethod interface {
	common.AuthMethod
	clientConfig() *ssh.ClientConfig
}

type PublicKeysCallback struct {
	user  string
	agent agent.Agent
}

func (a *PublicKeysCallback) Name() string {
	return "ssh-public-key-callback"
}

func (a *PublicKeysCallback) String() string {
	return "TODO"
}

func (a *PublicKeysCallback) clientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: a.user,
		Auth: []ssh.AuthMethod{ssh.PublicKeysCallback(a.agent.Signers)},
	}
}
