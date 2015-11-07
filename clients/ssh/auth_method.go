package ssh

import (
	"fmt"

	"golang.org/x/crypto/ssh"
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

// The names of the AuthMethod implementations. To be returned by the
// Name() method.
const (
	KeyboardInteractiveName = "ssh-keyboard-interactive"
	PasswordName            = "ssh-password"
	PasswordCallbackName    = "ssh-password-callback"
	PublicKeysName          = "ssh-public-keys"
	PublicKeysCallbackName  = "ssh-public-key-callback"
)

// PublicKeysCallback implements AuthMethod by storing an
// ssh.agent.Agent to act as a signer.
type PublicKeysCallback struct {
	user    string
	setAuth func() ([]ssh.Signer, error)
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
		Auth: []ssh.AuthMethod{ssh.PublicKeysCallback(a.setAuth)},
	}
}
