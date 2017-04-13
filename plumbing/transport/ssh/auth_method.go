package ssh

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var ErrEmptySSHAgentAddr = errors.New("SSH_AUTH_SOCK env variable is required")
var certChecker = &ssh.CertChecker{}

// AuthMethod is the interface all auth methods for the ssh client
// must implement. The ClientConfig method returns the ssh client
// configuration needed to establish an ssh connection.
type AuthMethod interface {
	ClientConfig() *ssh.ClientConfig
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
	User      string
	Challenge ssh.KeyboardInteractiveChallenge
}

func (a *KeyboardInteractive) Name() string {
	return KeyboardInteractiveName
}

func (a *KeyboardInteractive) String() string {
	return fmt.Sprintf("user: %s, name: %s", a.User, a.Name())
}

func (a *KeyboardInteractive) ClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: a.User,
		Auth: []ssh.AuthMethod{ssh.KeyboardInteractiveChallenge(a.Challenge)},
	}
}

// Password implements AuthMethod by using the given password.
// Requires a valid host key in ~/.ssh/known_hosts
type Password struct {
	User string
	Pass string
}

func (a *Password) Name() string {
	return PasswordName
}

func (a *Password) String() string {
	return fmt.Sprintf("user: %s, name: %s", a.User, a.Name())
}

func (a *Password) ClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            a.User,
		Auth:            []ssh.AuthMethod{ssh.Password(a.Pass)},
		HostKeyCallback: hostKeyChecker,
	}
}

// PasswordCallback implements AuthMethod by using a callback
// to fetch the password.
type PasswordCallback struct {
	User     string
	Callback func() (pass string, err error)
}

func (a *PasswordCallback) Name() string {
	return PasswordCallbackName
}

func (a *PasswordCallback) String() string {
	return fmt.Sprintf("user: %s, name: %s", a.User, a.Name())
}

func (a *PasswordCallback) ClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: a.User,
		Auth: []ssh.AuthMethod{ssh.PasswordCallback(a.Callback)},
	}
}

// PublicKeys implements AuthMethod by using the given
// key pairs. Requires a valid host key in ~/.ssh/known_hosts
type PublicKeys struct {
	User   string
	Signer ssh.Signer
}

func (a *PublicKeys) Name() string {
	return PublicKeysName
}

func (a *PublicKeys) String() string {
	return fmt.Sprintf("user: %s, name: %s", a.User, a.Name())
}

func (a *PublicKeys) ClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            a.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(a.Signer)},
		HostKeyCallback: hostKeyChecker,
	}
}

// PublicKeysCallback implements AuthMethod by asking a
// ssh.agent.Agent to act as a signer.
type PublicKeysCallback struct {
	User     string
	Callback func() (signers []ssh.Signer, err error)
}

func (a *PublicKeysCallback) Name() string {
	return PublicKeysCallbackName
}

func (a *PublicKeysCallback) String() string {
	return fmt.Sprintf("user: %s, name: %s", a.User, a.Name())
}

func (a *PublicKeysCallback) ClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            a.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeysCallback(a.Callback)},
		HostKeyCallback: hostKeyChecker,
	}
}

const DefaultSSHUsername = "git"

// NewSSHAgentAuth opens a pipe with the SSH agent and uses the pipe
// as the implementer of the public key callback function.
func NewSSHAgentAuth(user string) (*PublicKeysCallback, error) {
	if user == "" {
		user = DefaultSSHUsername
	}

	sshAgentAddr := os.Getenv("SSH_AUTH_SOCK")
	if sshAgentAddr == "" {
		return nil, ErrEmptySSHAgentAddr
	}

	pipe, err := net.Dial("unix", sshAgentAddr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to SSH agent: %q", err)
	}

	return &PublicKeysCallback{
		User:     user,
		Callback: agent.NewClient(pipe).Signers,
	}, nil
}

func knownHostHash(hostname string, salt64 string) (hash string, err error) {
	buffer, err := base64.StdEncoding.DecodeString(salt64)
	if err != nil {
		return hash, err
	}
	h := hmac.New(sha1.New, buffer)
	h.Write([]byte(hostname))
	hash = base64.StdEncoding.EncodeToString(h.Sum(nil))
	return
}

// Implements ssh.HostKeyCallback which is now required due to CVE-2017-3204
func hostKeyChecker(hostname string, remote net.Addr, key ssh.PublicKey) error {
	file, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		return fmt.Errorf("Host key verification: %s", err)
	}
	defer file.Close()

	// Remove standard port if given, add square brackets for non-standard ones
	hp := strings.Split(hostname, ":")
	if len(hp) == 2 {
		if hp[1] == "22" {
			hostname = hp[0]
		} else {
			hostname = "[" + hp[0] + "]:" + hp[1]
		}
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		marker, hosts, hostKey, _, _, err := ssh.ParseKnownHosts(scanner.Bytes())
		if err == io.EOF {
			continue
		}
		if err != nil {
			return fmt.Errorf("Host key verification: known_hosts parse error: %s", err)
		}
		if marker != "" {
			continue // @cert-authority or @revoked
		}
		if bytes.Compare(key.Marshal(), hostKey.Marshal()) == 0 {
			for _, host := range hosts {
				if len(host) > 1 && host[0:1] == "|" {
					parts := strings.Split(host, "|")
					if parts[1] != "1" {
						continue
					}
					hash, err := knownHostHash(hostname, parts[2])
					if err != nil {
						// If knownHostHash fails then ignore and continue
						continue
					}
					if hash == parts[3] {
						// Found a matching hashed hostname
						return nil
					}
				} else {
					if host == hostname {
						// Found a matching hostname
						return nil
					}
				}
			}
		}
	}
	return fmt.Errorf("Host key verification: no hostkey found for %s", hostname)
}
