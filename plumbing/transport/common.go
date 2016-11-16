// Package clients includes the implementation for different transport protocols
//
// go-git needs the packfile and the refs of the repo. The
// `NewUploadPackService` function returns an object that allows to
// download them.
//
// go-git supports HTTP and SSH (see `Protocols`) for downloading the packfile
// and the refs, but you can also install your own protocols (see
// `InstallProtocol` below).
//
// Each protocol has its own implementation of
// `NewUploadPackService`, but you should generally not use them
// directly, use this package's `NewUploadPackService` instead.
package transport

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"gopkg.in/src-d/go-git.v4/plumbing"
)

var (
	ErrRepositoryNotFound     = errors.New("repository not found")
	ErrAuthorizationRequired  = errors.New("authorization required")
	ErrEmptyUploadPackRequest = errors.New("empty git-upload-pack given")
	ErrInvalidAuthMethod      = errors.New("invalid auth method")
)

const (
	UploadPackServiceName = "git-upload-pack"
)

type Client interface {
	Connect() error
	Disconnect() error
	SetAuth(AuthMethod) error
	FetchPackService
}

// ClientFactory instantiates a Client with given endpoint.
type ClientFactory func(Endpoint) Client

type AuthMethod interface {
	fmt.Stringer
	Name() string
}

type Endpoint url.URL

var (
	isSchemeRegExp   = regexp.MustCompile("^[^:]+://")
	scpLikeUrlRegExp = regexp.MustCompile("^(?P<user>[^@]+@)?(?P<host>[^:]+):/?(?P<path>.+)$")
)

func NewEndpoint(endpoint string) (Endpoint, error) {
	endpoint = transformSCPLikeIfNeeded(endpoint)

	u, err := url.Parse(endpoint)
	if err != nil {
		return Endpoint{}, plumbing.NewPermanentError(err)
	}

	if !u.IsAbs() {
		return Endpoint{}, plumbing.NewPermanentError(fmt.Errorf(
			"invalid endpoint: %s", endpoint,
		))
	}

	return Endpoint(*u), nil
}

func (e *Endpoint) String() string {
	u := url.URL(*e)
	return u.String()
}

func transformSCPLikeIfNeeded(endpoint string) string {
	if !isSchemeRegExp.MatchString(endpoint) && scpLikeUrlRegExp.MatchString(endpoint) {
		m := scpLikeUrlRegExp.FindStringSubmatch(endpoint)
		return fmt.Sprintf("ssh://%s%s/%s", m[1], m[2], m[3])
	}

	return endpoint
}
