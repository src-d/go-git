// Package common contains interfaces and non-specific protocol entities
package common

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"

	"gopkg.in/src-d/go-git.v4/core"
	"gopkg.in/src-d/go-git.v4/formats/packp"
	"gopkg.in/src-d/go-git.v4/formats/packp/advrefs"
	"gopkg.in/src-d/go-git.v4/formats/packp/pktline"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

var (
	ErrRepositoryNotFound    = errors.New("repository not found")
	ErrAuthorizationRequired = errors.New("authorization required")
	ErrEmptyGitUploadPack    = errors.New("empty git-upload-pack given")
	ErrInvalidAuthMethod     = errors.New("invalid auth method")
)

const GitUploadPackServiceName = "git-upload-pack"

type GitUploadPackService interface {
	Connect() error
	SetAuth(AuthMethod) error
	Info() (*GitUploadPackInfo, error)
	Fetch(*GitUploadPackRequest) (io.ReadCloser, error)
	Disconnect() error
}

type AuthMethod interface {
	Name() string
	String() string
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
		return Endpoint{}, core.NewPermanentError(err)
	}

	if !u.IsAbs() {
		return Endpoint{}, core.NewPermanentError(fmt.Errorf(
			"invalid endpoint: %s", endpoint,
		))
	}

	return Endpoint(*u), nil
}

func transformSCPLikeIfNeeded(endpoint string) string {
	if !isSchemeRegExp.MatchString(endpoint) && scpLikeUrlRegExp.MatchString(endpoint) {
		m := scpLikeUrlRegExp.FindStringSubmatch(endpoint)
		return fmt.Sprintf("ssh://%s%s/%s", m[1], m[2], m[3])
	}

	return endpoint
}

func (e *Endpoint) String() string {
	u := url.URL(*e)
	return u.String()
}

type GitUploadPackInfo struct {
	Capabilities *packp.Capabilities
	Refs         memory.ReferenceStorage
}

func NewGitUploadPackInfo() *GitUploadPackInfo {
	return &GitUploadPackInfo{Capabilities: packp.NewCapabilities()}
}

func (i *GitUploadPackInfo) Decode(r io.Reader) error {
	d := advrefs.NewDecoder(r)
	ar := advrefs.New()
	if err := d.Decode(ar); err != nil {
		if err == advrefs.ErrEmpty {
			return core.NewPermanentError(err)
		}
		return core.NewUnexpectedError(err)
	}

	i.Capabilities = ar.Caps
	i.Refs = make(memory.ReferenceStorage, 0)
	for name, hash := range ar.Refs {
		ref := core.NewReferenceFromStrings(name, hash.String())
		i.Refs.Set(ref)
	}

	if hasHeadSymref(ar) {
		target := i.Capabilities.SymbolicReference(string(core.HEAD))
		head := core.NewSymbolicReference(core.HEAD, core.ReferenceName(target))
		i.Refs.Set(head)
	}

	return nil
}

func hasHeadSymref(ar *advrefs.AdvRefs) bool {
	return ar.Caps.Supports("symref") && ar.Head != nil
}

func (i *GitUploadPackInfo) Head() *core.Reference {
	ref, _ := core.ResolveReference(i.Refs, core.HEAD)
	return ref
}

func (i *GitUploadPackInfo) String() string {
	return string(i.Bytes())
}

func (i *GitUploadPackInfo) Bytes() []byte {
	p := pktline.New()
	_ = p.AddString("# service=git-upload-pack\n")
	// inserting a flush-pkt here violates the protocol spec, but some
	// servers do it, like Github.com
	p.AddFlush()

	firstLine := fmt.Sprintf("%s HEAD\x00%s\n", i.Head().Hash(), i.Capabilities.String())
	_ = p.AddString(firstLine)

	for _, ref := range i.Refs {
		if ref.Type() != core.HashReference {
			continue
		}

		ref := fmt.Sprintf("%s %s\n", ref.Hash(), ref.Name())
		_ = p.AddString(ref)
	}

	p.AddFlush()
	b, _ := ioutil.ReadAll(p)

	return b
}

type GitUploadPackRequest struct {
	Wants []core.Hash
	Haves []core.Hash
	Depth int
}

func (r *GitUploadPackRequest) Want(h ...core.Hash) {
	r.Wants = append(r.Wants, h...)
}

func (r *GitUploadPackRequest) Have(h ...core.Hash) {
	r.Haves = append(r.Haves, h...)
}

func (r *GitUploadPackRequest) String() string {
	b, _ := ioutil.ReadAll(r.Reader())
	return string(b)
}

func (r *GitUploadPackRequest) Reader() *strings.Reader {
	p := pktline.New()

	for _, want := range r.Wants {
		_ = p.AddString(fmt.Sprintf("want %s\n", want))
	}

	for _, have := range r.Haves {
		_ = p.AddString(fmt.Sprintf("have %s\n", have))
	}

	if r.Depth != 0 {
		_ = p.AddString(fmt.Sprintf("deepen %d\n", r.Depth))
	}

	p.AddFlush()
	_ = p.AddString("done\n")

	b, _ := ioutil.ReadAll(p)

	return strings.NewReader(string(b))
}
