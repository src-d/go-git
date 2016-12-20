// Package server implements the git server protocol. For most use cases, the
// transport-specific implementations should be used.
package server

import (
	"errors"
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packfile"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp/capability"
	"gopkg.in/src-d/go-git.v4/plumbing/revlist"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

// Handler is server-side a protocol implementation.
type Handler interface {
	// NewUploadPackSession starts a git-upload-pack session for a given
	// repository.
	NewUploadPackSession(storer.Storer) (UploadPackSession, error)
	// NewReceivePackSession starts a git-receive-pack session for a given
	// repository.
	NewReceivePackSession(storer.Storer) (ReceivePackSession, error)
}

type Session interface {
	AdvertisedReferences() (*packp.AdvRefs, error)
}

// UploadPackSession is a git-upload-pack session.
type UploadPackSession interface {
	Session
	UploadPack(*packp.UploadPackRequest) (*packp.UploadPackResponse, error)
}

// UploadPackSession is a git-receive-pack session.
type ReceivePackSession interface {
	Session
	ReceivePack(*packp.ReferenceUpdateRequest) (*packp.ReportStatus, error)
}

// DefaultHandler is the default server handler. Use this unless you are
// creating your own server implementation.
var DefaultHandler = NewHandler()

type handler struct{}

// NewHandler creates a new server handler.
// Use DefaultHandler instead.
func NewHandler() Handler {
	return &handler{}
}

func (h *handler) NewUploadPackSession(s storer.Storer) (UploadPackSession, error) {
	return &upSession{
		session: session{storer: s},
	}, nil
}

func (h *handler) NewReceivePackSession(s storer.Storer) (ReceivePackSession, error) {
	return &rpSession{
		session:   session{storer: s},
		cmdStatus: map[plumbing.ReferenceName]error{},
	}, nil
}

type session struct {
	storer  storer.Storer
	advRefs *packp.AdvRefs
}

func (s *session) checkSupportedCapabilities(cl *capability.List) error {
	for _, c := range cl.All() {
		if !s.advRefs.Capabilities.Supports(c) {
			return fmt.Errorf("unsupported capability: %s", c)
		}
	}

	return nil
}

type upSession struct {
	session
}

func (s *upSession) AdvertisedReferences() (*packp.AdvRefs, error) {
	ar := packp.NewAdvRefs()

	if err := s.setSupportedCapabilities(ar.Capabilities); err != nil {
		return nil, err
	}

	if err := setReferences(s.storer, ar); err != nil {
		return nil, err
	}

	if err := setHEAD(s.storer, ar); err != nil {
		return nil, err
	}

	s.advRefs = ar

	return ar, nil
}

func (s *upSession) UploadPack(req *packp.UploadPackRequest) (*packp.UploadPackResponse, error) {
	if req.IsEmpty() {
		return nil, transport.ErrEmptyUploadPackRequest
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	if err := s.checkSupportedCapabilities(req.Capabilities); err != nil {
		return nil, err
	}

	if len(req.Shallows) > 0 {
		return nil, fmt.Errorf("shallow not supported")
	}

	objs, err := s.objectsToUpload(req)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	e := packfile.NewEncoder(pw, s.storer, false)
	go func() {
		_, err := e.Encode(objs)
		pw.CloseWithError(err)
	}()

	resp := packp.NewUploadPackResponse(req)
	resp.Packfile = pr

	return resp, nil
}

func (s *upSession) objectsToUpload(req *packp.UploadPackRequest) ([]plumbing.Hash, error) {
	commits, err := s.commitsToUpload(req.Wants)
	if err != nil {
		return nil, err
	}

	return revlist.Objects(s.storer, commits, req.Haves)
}

func (s *upSession) commitsToUpload(wants []plumbing.Hash) ([]*object.Commit, error) {
	var commits []*object.Commit
	for _, h := range wants {
		c, err := object.GetCommit(s.storer, h)
		if err != nil {
			return nil, err
		}

		commits = append(commits, c)
	}

	return commits, nil
}

func (*upSession) setSupportedCapabilities(c *capability.List) error {
	if err := c.Set(capability.Agent, capability.DefaultAgent); err != nil {
		return err
	}

	if err := c.Set(capability.OFSDelta); err != nil {
		return err
	}

	return nil
}

type rpSession struct {
	session
	cmdStatus map[plumbing.ReferenceName]error
	firstErr  error
	unpackErr error
	cap       *capability.List
}

func (s *rpSession) AdvertisedReferences() (*packp.AdvRefs, error) {
	ar := packp.NewAdvRefs()

	if err := s.setSupportedCapabilities(ar.Capabilities); err != nil {
		return nil, err
	}

	if err := setReferences(s.storer, ar); err != nil {
		return nil, err
	}

	if err := setHEAD(s.storer, ar); err != nil {
		return nil, err
	}

	s.advRefs = ar

	return ar, nil
}

var (
	ErrReferenceAlreadyExists = errors.New("reference already exists")
	ErrReferenceDidNotExist   = errors.New("reference did not exist")
)

func (s *rpSession) ReceivePack(req *packp.ReferenceUpdateRequest) (*packp.ReportStatus, error) {
	if err := s.checkSupportedCapabilities(req.Capabilities); err != nil {
		return nil, err
	}

	s.cap = req.Capabilities

	//TODO: Implement 'atomic' update of references.

	if err := s.writePackfile(req.Packfile); err != nil {
		s.unpackErr = err
		s.firstErr = err
		return s.reportStatus(), err
	}

	updatedRefs := s.updatedReferences(req)

	if s.cap.Supports(capability.Atomic) && s.firstErr != nil {
		//TODO: add support for 'atomic' once we have reference
		//      transactions, currently we do not announce it.
		rs := s.reportStatus()
		for _, cs := range rs.CommandStatuses {
			if cs.Error() == nil {
				cs.Status = ""
			}
		}
	}

	for name, ref := range updatedRefs {
		//TODO: add support for 'delete-refs' once we can delete
		//      references, currently we do not announce it.
		err := s.storer.SetReference(ref)
		s.setStatus(name, err)
	}

	return s.reportStatus(), s.firstErr
}

func (s *rpSession) updatedReferences(req *packp.ReferenceUpdateRequest) map[plumbing.ReferenceName]*plumbing.Reference {
	refs := map[plumbing.ReferenceName]*plumbing.Reference{}
	for _, cmd := range req.Commands {
		exists, err := referenceExists(s.storer, cmd.Name)
		if err != nil {
			s.setStatus(cmd.Name, err)
			continue
		}

		switch cmd.Action() {
		case packp.Create:
			if exists {
				s.setStatus(cmd.Name, ErrReferenceAlreadyExists)
				continue
			}

			ref := plumbing.NewHashReference(cmd.Name, cmd.New)
			refs[ref.Name()] = ref
		case packp.Delete:
			if !exists {
				s.setStatus(cmd.Name, ErrReferenceDidNotExist)
				continue
			}

			if !s.cap.Supports(capability.DeleteRefs) {
				s.setStatus(cmd.Name, fmt.Errorf("delete not supported"))
				continue
			}

			refs[cmd.Name] = nil
		case packp.Update:
			if !exists {
				s.setStatus(cmd.Name, ErrReferenceDidNotExist)
				continue
			}

			if err != nil {
				s.setStatus(cmd.Name, err)
				continue
			}

			ref := plumbing.NewHashReference(cmd.Name, cmd.New)
			refs[ref.Name()] = ref
		}
	}

	return refs
}

func (s *rpSession) failAtomicUpdate() (*packp.ReportStatus, error) {
	rs := s.reportStatus()
	for _, cs := range rs.CommandStatuses {
		if cs.Error() == nil {
			cs.Status = "atomic updated"
		}
	}

	return rs, s.firstErr
}

func (s *rpSession) writePackfile(r io.ReadCloser) error {
	if r == nil {
		return nil
	}

	if err := packfile.UpdateObjectStorage(s.storer, r); err != nil {
		_ = r.Close()
		return err
	}

	return r.Close()
}

func (s *rpSession) setStatus(ref plumbing.ReferenceName, err error) {
	s.cmdStatus[ref] = err
	if s.firstErr != nil && err != nil {
		s.firstErr = err
	}
}

func (s *rpSession) reportStatus() *packp.ReportStatus {
	if !s.cap.Supports(capability.ReportStatus) {
		return nil
	}

	rs := packp.NewReportStatus()
	rs.UnpackStatus = "ok"

	if s.unpackErr != nil {
		rs.UnpackStatus = s.unpackErr.Error()
	}

	if s.cmdStatus == nil {
		return rs
	}

	for ref, err := range s.cmdStatus {
		msg := "ok"
		if err != nil {
			msg = err.Error()
		}
		status := &packp.CommandStatus{
			ReferenceName: ref,
			Status:        msg,
		}
		rs.CommandStatuses = append(rs.CommandStatuses, status)
	}

	return rs
}

func (*rpSession) setSupportedCapabilities(c *capability.List) error {
	if err := c.Set(capability.Agent, capability.DefaultAgent); err != nil {
		return err
	}

	if err := c.Set(capability.OFSDelta); err != nil {
		return err
	}

	return c.Set(capability.ReportStatus)
}

func setHEAD(s storer.Storer, ar *packp.AdvRefs) error {
	ref, err := s.Reference(plumbing.HEAD)
	if err == plumbing.ErrReferenceNotFound {
		return nil
	}

	if err != nil {
		return err
	}

	if ref.Type() == plumbing.SymbolicReference {
		if err := ar.AddReference(ref); err != nil {
			return nil
		}

		ref, err = storer.ResolveReference(s, ref.Target())
		if err == plumbing.ErrReferenceNotFound {
			return nil
		}

		if err != nil {
			return err
		}
	}

	if ref.Type() != plumbing.HashReference {
		return plumbing.ErrInvalidType
	}

	h := ref.Hash()
	ar.Head = &h

	return nil
}

//TODO: add peeled references.
func setReferences(s storer.Storer, ar *packp.AdvRefs) error {
	iter, err := s.IterReferences()
	if err != nil {
		return err
	}

	return iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() != plumbing.HashReference {
			return nil
		}

		ar.References[ref.Name().String()] = ref.Hash()
		return nil
	})
}

func referenceExists(s storer.ReferenceStorer, n plumbing.ReferenceName) (bool, error) {
	_, err := s.Reference(n)
	if err == plumbing.ErrReferenceNotFound {
		return false, nil
	}

	return err == nil, err
}
