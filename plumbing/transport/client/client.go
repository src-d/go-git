package client

import (
	"fmt"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

// Protocols are the protocols supported by default.
var Protocols = map[string]transport.ClientFactory{
	"http":  http.NewClient,
	"https": http.NewClient,
	"ssh":   ssh.NewClient,
}

// InstallProtocol adds or modifies an existing protocol.
func InstallProtocol(scheme string, f transport.ClientFactory) {
	Protocols[scheme] = f
}

// NewClient returns the appropriate client among of the set of known protocols:
// HTTP, SSH. See `InstallProtocol` to add or modify protocols.
func NewClient(endpoint transport.Endpoint) (transport.Client, error) {
	f, ok := Protocols[endpoint.Scheme]
	if !ok {
		return nil, fmt.Errorf("unsupported scheme %q", endpoint.Scheme)
	}

	return f(endpoint), nil
}
