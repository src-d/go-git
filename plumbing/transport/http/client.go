// Package http implements a HTTP client for go-git.
package http

import (
	"fmt"
	"net/http"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

type Client struct {
	c    *http.Client
	ep   transport.Endpoint
	auth AuthMethod
}

func NewClient(ep transport.Endpoint) transport.Client {
	return newClient(nil, ep)
}

// NewClientFactory creates a http client factory with a customizable client
// See `InstallProtocol` to install and override default http client.
// Unless a properly initialized client is given, it will fall back into
// `http.DefaultClient`.
func NewClientFactory(c *http.Client) transport.ClientFactory {
	return func(ep transport.Endpoint) transport.Client {
		return newClient(c, ep)
	}
}

func newClient(c *http.Client, ep transport.Endpoint) transport.Client {
	if c == nil {
		c = http.DefaultClient
	}

	return &Client{
		c:    c,
		ep:   ep,
		auth: basicAuthFromEndpoint(ep),
	}
}

// Connect has no effect.
func (s *Client) Connect() error {
	return nil
}

// Disconnect has no effect.
func (*Client) Disconnect() error {
	return nil
}

// SetAuth sets the AuthMethod
func (s *Client) SetAuth(auth transport.AuthMethod) error {
	httpAuth, ok := auth.(AuthMethod)
	if !ok {
		return transport.ErrInvalidAuthMethod
	}

	s.auth = httpAuth
	return nil
}

func basicAuthFromEndpoint(ep transport.Endpoint) AuthMethod {
	info := ep.User
	if info == nil {
		return nil
	}

	p, ok := info.Password()
	if !ok {
		return nil
	}

	u := info.Username()
	return NewBasicAuth(u, p)
}

// AuthMethod is concrete implementation of common.AuthMethod for HTTP services
type AuthMethod interface {
	transport.AuthMethod
	setAuth(r *http.Request)
}

// BasicAuth represent a HTTP basic auth
type BasicAuth struct {
	username, password string
}

// NewBasicAuth returns a BasicAuth base on the given user and password
func NewBasicAuth(username, password string) *BasicAuth {
	return &BasicAuth{username, password}
}

func (a *BasicAuth) setAuth(r *http.Request) {
	r.SetBasicAuth(a.username, a.password)
}

// Name is name of the auth
func (a *BasicAuth) Name() string {
	return "http-basic-auth"
}

func (a *BasicAuth) String() string {
	masked := "*******"
	if a.password == "" {
		masked = "<empty>"
	}

	return fmt.Sprintf("%s - %s:%s", a.Name(), a.username, masked)
}

// Err is a dedicated error to return errors based on status code
type Err struct {
	Response *http.Response
}

// NewErr returns a new Err based on a http response
func NewErr(r *http.Response) error {
	if r.StatusCode >= http.StatusOK && r.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	switch r.StatusCode {
	case http.StatusUnauthorized:
		return transport.ErrAuthorizationRequired
	case http.StatusNotFound:
		return transport.ErrRepositoryNotFound
	}

	return plumbing.NewUnexpectedError(&Err{r})
}

// StatusCode returns the status code of the response
func (e *Err) StatusCode() int {
	return e.Response.StatusCode
}

func (e *Err) Error() string {
	return fmt.Sprintf("unexpected requesting %q status code: %d",
		e.Response.Request.URL, e.Response.StatusCode,
	)
}
