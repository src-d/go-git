// Package transport includes the implementation for different transport
// protocols.
//
// `Client` can be used to fetch and send packfiles to a git server.
// The `client` package provides higher level functions to instantiate the
// appropriate `Client` based on the repository URL.
//
// go-git supports HTTP and SSH (see `Protocols`), but you can also install
// your own protocols (see the `client` package).
//
// Each protocol has its own implementation of `Client`, but you should
// generally not use them directly, use `client.NewClient` instead.
package transport

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp/capability"
)

var (
	ErrRepositoryNotFound     = errors.New("repository not found")
	ErrEmptyRemoteRepository  = errors.New("remote repository is empty")
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrAuthorizationFailed    = errors.New("authorization failed")
	ErrEmptyUploadPackRequest = errors.New("empty git-upload-pack given")
	ErrInvalidAuthMethod      = errors.New("invalid auth method")
	ErrAlreadyConnected       = errors.New("session already established")
)

const (
	UploadPackServiceName  = "git-upload-pack"
	ReceivePackServiceName = "git-receive-pack"
)

// Transport can initiate git-upload-pack and git-receive-pack processes.
// It is implemented both by the client and the server, making this a RPC.
type Transport interface {
	// NewUploadPackSession starts a git-upload-pack session for an endpoint.
	NewUploadPackSession(Endpoint, AuthMethod) (UploadPackSession, error)
	// NewReceivePackSession starts a git-receive-pack session for an endpoint.
	NewReceivePackSession(Endpoint, AuthMethod) (ReceivePackSession, error)
}

type Session interface {
	// AdvertisedReferences retrieves the advertised references for a
	// repository.
	// If the repository does not exist, returns ErrRepositoryNotFound.
	// If the repository exists, but is empty, returns ErrEmptyRemoteRepository.
	AdvertisedReferences() (*packp.AdvRefs, error)
	io.Closer
}

type AuthMethod interface {
	fmt.Stringer
	Name() string
}

// UploadPackSession represents a git-upload-pack session.
// A git-upload-pack session has two steps: reference discovery
// (AdvertisedReferences) and uploading pack (UploadPack).
type UploadPackSession interface {
	Session
	// UploadPack takes a git-upload-pack request and returns a response,
	// including a packfile. Don't be confused by terminology, the client
	// side of a git-upload-pack is called git-fetch-pack, although here
	// the same interface is used to make it RPC-like.
	UploadPack(*packp.UploadPackRequest) (*packp.UploadPackResponse, error)
}

// ReceivePackSession represents a git-receive-pack session.
// A git-receive-pack session has two steps: reference discovery
// (AdvertisedReferences) and receiving pack (ReceivePack).
// In that order.
type ReceivePackSession interface {
	Session
	// ReceivePack sends an update references request and a packfile
	// reader and returns a ReportStatus and error. Don't be confused by
	// terminology, the client side of a git-receive-pack is called
	// git-send-pack, although here the same interface is used to make it
	// RPC-like.
	ReceivePack(*packp.ReferenceUpdateRequest) (*packp.ReportStatus, error)
}

// Endpoint represents a Git URL in any supported protocol.
type Endpoint interface {
	// Protocol returns the protocol (e.g. git, https, file). It should never
	// return the empty string.
	Protocol() string
	// User returns the user or an empty string if none is given.
	User() string
	// Password returns the password or an empty string if none is given.
	Password() string
	// Host returns the host or an empty string if none is given.
	Host() string
	// Port returns the port or 0 if there is no port or a default should be
	// used.
	Port() int
	// Path returns the repository path.
	Path() string
	// String returns a string representation of the Git URL.
	String() string
}

func NewEndpoint(endpoint string) (Endpoint, error) {
	if e, ok := parseSCPLike(endpoint); ok {
		return e, nil
	}

	if e, ok := parseFile(endpoint); ok {
		return e, nil
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, plumbing.NewPermanentError(err)
	}

	if !u.IsAbs() {
		return nil, plumbing.NewPermanentError(fmt.Errorf(
			"invalid endpoint: %s", endpoint,
		))
	}

	return URLEndpoint{u}, nil
}

type URLEndpoint struct {
	*url.URL
}

func (e URLEndpoint) Protocol() string { return e.URL.Scheme }
func (e URLEndpoint) Host() string     { return e.URL.Hostname() }

func (e URLEndpoint) User() string {
	if e.URL.User == nil {
		return ""
	}

	return e.URL.User.Username()
}

func (e URLEndpoint) Password() string {
	if e.URL.User == nil {
		return ""
	}

	p, _ := e.URL.User.Password()
	return p
}

func (e URLEndpoint) Port() int {
	p := e.URL.Port()
	if p == "" {
		return 0
	}

	i, err := strconv.Atoi(e.URL.Port())
	if err != nil {
		return 0
	}

	return i
}

func (e URLEndpoint) Path() string {
	var res string = e.URL.Path
	if e.URL.RawQuery != "" {
		res += "?" + e.URL.RawQuery
	}

	if e.URL.Fragment != "" {
		res += "#" + e.URL.Fragment
	}

	return res
}

type SCPEndpoint struct {
	// Renamed to avoid collision with methods.
	Username string
	Hostname string
	Pathname string
}

func (e *SCPEndpoint) Protocol() string { return "ssh" }
func (e *SCPEndpoint) User() string     { return e.Username }
func (e *SCPEndpoint) Password() string { return "" }
func (e *SCPEndpoint) Host() string     { return e.Hostname }
func (e *SCPEndpoint) Port() int        { return 22 }
func (e *SCPEndpoint) Path() string     { return e.Pathname }

func (e *SCPEndpoint) String() string {
	var user string
	if e.Username != "" {
		user = fmt.Sprintf("%s@", e.Username)
	}

	return fmt.Sprintf("%s%s:%s", user, e.Hostname, e.Pathname)
}

type FileEndpoint struct {
	Pathname string
}

func (e *FileEndpoint) Protocol() string { return "file" }
func (e *FileEndpoint) User() string     { return "" }
func (e *FileEndpoint) Password() string { return "" }
func (e *FileEndpoint) Host() string     { return "" }
func (e *FileEndpoint) Port() int        { return 0 }
func (e *FileEndpoint) Path() string     { return e.Pathname }
func (e *FileEndpoint) String() string   { return e.Pathname }

var (
	isSchemeRegExp   = regexp.MustCompile(`^[^:]+://`)
	scpLikeUrlRegExp = regexp.MustCompile(`^(?:(?P<user>[^@]+)@)?(?P<host>[^:\s]+):(?P<path>[^\\].*)$`)
)

func parseSCPLike(endpoint string) (Endpoint, bool) {
	if isSchemeRegExp.MatchString(endpoint) || !scpLikeUrlRegExp.MatchString(endpoint) {
		return nil, false
	}

	m := scpLikeUrlRegExp.FindStringSubmatch(endpoint)
	return &SCPEndpoint{
		Username: m[1],
		Hostname: m[2],
		Pathname: m[3],
	}, true
}

func parseFile(endpoint string) (Endpoint, bool) {
	if isSchemeRegExp.MatchString(endpoint) {
		return nil, false
	}

	return &FileEndpoint{Pathname: endpoint}, true
}

// UnsupportedCapabilities are the capabilities not supported by any client
// implementation
var UnsupportedCapabilities = []capability.Capability{
	capability.MultiACK,
	capability.MultiACKDetailed,
	capability.ThinPack,
}

// FilterUnsupportedCapabilities it filter out all the UnsupportedCapabilities
// from a capability.List, the intended usage is on the client implementation
// to filter the capabilities from an AdvRefs message.
func FilterUnsupportedCapabilities(list *capability.List) {
	for _, c := range UnsupportedCapabilities {
		list.Delete(c)
	}
}
