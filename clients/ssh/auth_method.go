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
// Name() method. Most git servers only allow PublicKeysName and
// PublicKeysCallbackName.
const (
	KeyboardInteractiveName = "ssh-keyboard-interactive"
	PasswordName            = "ssh-password"
	PasswordCallbackName    = "ssh-password-callback"
	PublicKeysName          = "ssh-public-keys"
	PublicKeysCallbackName  = "ssh-public-key-callback"
)

// KeyboardInteractive implements AuthMethod by using a
// prompt/response sequence controlled by the server.
type KeyboardInteractive struct {
	user      string
	challenge ssh.KeyboardInteractiveChallenge
}

// Name returns KeyboardInteractive.
func (a *KeyboardInteractive) Name() string {
	return KeyboardInteractiveName
}

func (a *KeyboardInteractive) String() string {
	return fmt.Sprintf("user: %s, name: %s", a.user, a.Name())
}

func (a *KeyboardInteractive) clientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: a.user,
		Auth: []ssh.AuthMethod{ssh.KeyboardInteractiveChallenge(a.challenge)},
	}
}

// Password implements AuthMethod by using the given password.
type Password struct {
	user string
	pass string
}

// Name returns Password.
func (a *Password) Name() string {
	return PasswordName
}

func (a *Password) String() string {
	return fmt.Sprintf("user: %s, name: %s", a.user, a.Name())
}

func (a *Password) clientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: a.user,
		Auth: []ssh.AuthMethod{ssh.Password(a.pass)},
	}
}

// PasswordCallback implements AuthMethod by using a callback
// to fetch the password.
type PasswordCallback struct {
	user     string
	callback func() (pass string, err error)
}

// Name returns PasswordCallback.
func (a *PasswordCallback) Name() string {
	return PasswordCallbackName
}

func (a *PasswordCallback) String() string {
	return fmt.Sprintf("user: %s, name: %s", a.user, a.Name())
}

func (a *PasswordCallback) clientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: a.user,
		Auth: []ssh.AuthMethod{ssh.PasswordCallback(a.callback)},
	}
}

// PublicKeys implements AuthMethod by using the given
// key pairs.
type PublicKeys struct {
	user   string
	signer ssh.Signer
}

// Name returns PublicKeys.
func (a *PublicKeys) Name() string {
	return PublicKeysName
}

func (a *PublicKeys) String() string {
	return fmt.Sprintf("user: %s, name: %s", a.user, a.Name())
}

func (a *PublicKeys) clientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: a.user,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(a.signer)},
	}
}

// PublicKeysCallback implements AuthMethod by storing an
// ssh.agent.Agent to act as a signer.
type PublicKeysCallback struct {
	user     string
	callback func() (signers []ssh.Signer, err error)
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
		Auth: []ssh.AuthMethod{ssh.PublicKeysCallback(a.callback)},
	}
}
