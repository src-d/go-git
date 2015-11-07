package ssh

import (
	"fmt"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"gopkg.in/src-d/go-git.v2/clients/common"
)

// AuthMethod is the interface all auth methods for the ssh client
// must implement. The clientConfig method returns the ssh client
// configuration needed to establish an ssh connection.
//
// Current implementations:
// - PublicKeysCallback
type AuthMethod interface {
	common.AuthMethod
	clientConfig() *ssh.ClientConfig
}

// The names of the current AuthMethod implementations
const (
	PublicKeysCallbackName = "ssh-public-key-callback"
)

// PublicKeysCallback implements AuthMethod by storing an
// ssh.agent.Agent to act as a signer.
type PublicKeysCallback struct {
	user  string
	agent agent.Agent
}

// Name returns PublicKeysCallback.
func (a *PublicKeysCallback) Name() string {
	return PublicKeysCallbackName
}

func (a *PublicKeysCallback) String() string {
	return fmt.Sprintf("user: %s, name: %s", a.user, a.Name())
}

func (a *PublicKeysCallback) clientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: a.user,
		Auth: []ssh.AuthMethod{ssh.PublicKeysCallback(a.agent.Signers)},
	}
}
