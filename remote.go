package git

import (
	"errors"
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packfile"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp/capability"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp/sideband"
	"gopkg.in/src-d/go-git.v4/plumbing/revlist"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/client"
	"gopkg.in/src-d/go-git.v4/storage"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

var NoErrAlreadyUpToDate = errors.New("already up-to-date")

// Remote represents a connection to a remote repository.
type Remote struct {
	c *config.RemoteConfig
	s storage.Storer
}

func newRemote(s storage.Storer, c *config.RemoteConfig) *Remote {
	return &Remote{s: s, c: c}
}

// Config returns the RemoteConfig object used to instantiate this Remote.
func (r *Remote) Config() *config.RemoteConfig {
	return r.c
}

func (r *Remote) String() string {
	fetch := r.c.URL
	push := r.c.URL

	return fmt.Sprintf("%s\t%s (fetch)\n%[1]s\t%[3]s (push)", r.c.Name, fetch, push)
}

// Fetch fetches references from the remote to the local repository.
// Returns nil if the operation is successful, NoErrAlreadyUpToDate if there are
// no changes to be fetched, or an error.
func (r *Remote) Fetch(o *FetchOptions) error {
	_, err := r.fetch(o)
	return err
}

// Push performs a push to the remote. Returns NoErrAlreadyUpToDate if the
// remote was already up-to-date.
func (r *Remote) Push(o *PushOptions) (err error) {
	// TODO: Support deletes.
	// TODO: Sideband support

	if o.RemoteName == "" {
		o.RemoteName = r.c.Name
	}

	if err := o.Validate(); err != nil {
		return err
	}

	if o.RemoteName != r.c.Name {
		return fmt.Errorf("remote names don't match: %s != %s", o.RemoteName, r.c.Name)
	}

	s, err := newSendPackSession(r.c.URL, o.Auth)
	if err != nil {
		return err
	}

	ar, err := s.AdvertisedReferences()
	if err != nil {
		return err
	}

	remoteRefs, err := ar.AllReferences()
	if err != nil {
		return err
	}

	req := packp.NewReferenceUpdateRequestFromCapabilities(ar.Capabilities)
	if err := r.addReferencesToUpdate(o.RefSpecs, remoteRefs, req); err != nil {
		return err
	}

	if len(req.Commands) == 0 {
		return NoErrAlreadyUpToDate
	}

	objects, err := objectsToPush(req.Commands)
	if err != nil {
		return err
	}

	haves, err := referencesToHashes(remoteRefs)
	if err != nil {
		return err
	}

	hashesToPush, err := revlist.Objects(r.s, objects, haves)
	if err != nil {
		return err
	}

	rs, err := pushHashes(s, r.s, req, hashesToPush)
	if err != nil {
		return err
	}

	return rs.Error()
}

func (r *Remote) fetch(o *FetchOptions) (refs storer.ReferenceStorer, err error) {
	if o.RemoteName == "" {
		o.RemoteName = r.c.Name
	}

	if err := o.Validate(); err != nil {
		return nil, err
	}

	if len(o.RefSpecs) == 0 {
		o.RefSpecs = r.c.Fetch
	}

	s, err := newUploadPackSession(r.c.URL, o.Auth)
	if err != nil {
		return nil, err
	}

	defer ioutil.CheckClose(s, &err)

	ar, err := s.AdvertisedReferences()
	if err != nil {
		return nil, err
	}

	req, err := r.newUploadPackRequest(o, ar)
	if err != nil {
		return nil, err
	}

	remoteRefs, err := ar.AllReferences()
	if err != nil {
		return nil, err
	}

	req.Wants, err = getWants(o.RefSpecs, r.s, remoteRefs)
	if len(req.Wants) == 0 {
		return remoteRefs, NoErrAlreadyUpToDate
	}

	req.Haves, err = getHaves(r.s)
	if err != nil {
		return nil, err
	}

	if err := r.fetchPack(o, s, req); err != nil {
		return nil, err
	}

	if err := r.updateLocalReferenceStorage(o.RefSpecs, remoteRefs); err != nil {
		return nil, err
	}

	return remoteRefs, err
}

func newUploadPackSession(url string, auth transport.AuthMethod) (transport.UploadPackSession, error) {
	c, ep, err := newClient(url)
	if err != nil {
		return nil, err
	}

	return c.NewUploadPackSession(ep, auth)
}

func newSendPackSession(url string, auth transport.AuthMethod) (transport.ReceivePackSession, error) {
	c, ep, err := newClient(url)
	if err != nil {
		return nil, err
	}

	return c.NewReceivePackSession(ep, auth)
}

func newClient(url string) (transport.Transport, transport.Endpoint, error) {
	ep, err := transport.NewEndpoint(url)
	if err != nil {
		return nil, nil, err
	}

	c, err := client.NewClient(ep)
	if err != nil {
		return nil, nil, err
	}

	return c, ep, err
}

func (r *Remote) fetchPack(o *FetchOptions, s transport.UploadPackSession,
	req *packp.UploadPackRequest) (err error) {

	reader, err := s.UploadPack(req)
	if err != nil {
		return err
	}

	defer ioutil.CheckClose(reader, &err)

	if err := r.updateShallow(o, reader); err != nil {
		return err
	}

	if err = packfile.UpdateObjectStorage(r.s,
		buildSidebandIfSupported(req.Capabilities, reader, o.Progress),
	); err != nil {
		return err
	}

	return err
}

func (r *Remote) addReferencesToUpdate(refspecs []config.RefSpec,
	remoteRefs storer.ReferenceStorer,
	req *packp.ReferenceUpdateRequest) error {

	for _, rs := range refspecs {
		iter, err := r.s.IterReferences()
		if err != nil {
			return err
		}

		err = iter.ForEach(func(ref *plumbing.Reference) error {
			return r.addReferenceIfRefSpecMatches(
				rs, remoteRefs, ref, req,
			)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Remote) addReferenceIfRefSpecMatches(rs config.RefSpec,
	remoteRefs storer.ReferenceStorer, localRef *plumbing.Reference,
	req *packp.ReferenceUpdateRequest) error {

	if localRef.Type() != plumbing.HashReference {
		return nil
	}

	if !rs.Match(localRef.Name()) {
		return nil
	}

	cmd := &packp.Command{
		Name: rs.Dst(localRef.Name()),
		Old:  plumbing.ZeroHash,
		New:  localRef.Hash(),
	}

	remoteRef, err := remoteRefs.Reference(cmd.Name)
	if err == nil {
		if remoteRef.Type() != plumbing.HashReference {
			//TODO: check actual git behavior here
			return nil
		}

		cmd.Old = remoteRef.Hash()
	} else if err != plumbing.ErrReferenceNotFound {
		return err
	}

	if cmd.Old == cmd.New {
		return nil
	}

	if !rs.IsForceUpdate() {
		if err := checkFastForwardUpdate(r.s, remoteRefs, cmd); err != nil {
			return err
		}
	}

	req.Commands = append(req.Commands, cmd)
	return nil
}

func getHaves(localRefs storer.ReferenceStorer) ([]plumbing.Hash, error) {
	iter, err := localRefs.IterReferences()
	if err != nil {
		return nil, err
	}

	var haves []plumbing.Hash
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() != plumbing.HashReference {
			return nil
		}

		haves = append(haves, ref.Hash())
		return nil
	})
	if err != nil {
		return nil, err
	}

	return haves, nil
}

func getWants(
	spec []config.RefSpec, localStorer storage.Storer, remoteRefs storer.ReferenceStorer,
) ([]plumbing.Hash, error) {
	wantTags := true
	for _, s := range spec {
		if !s.IsWildcard() {
			wantTags = false
			break
		}
	}

	tags := []plumbing.Hash{}
	localIter, err := localStorer.IterReferences()
	if err != nil {
		return nil, err
	}

	localIter.ForEach(func(ref *plumbing.Reference) error {
		if !ref.IsTag() {
			return nil
		}

		tags = append(tags, ref.Hash())
		return nil
	})

	iter, err := remoteRefs.IterReferences()
	if err != nil {
		return nil, err
	}

	wants := map[plumbing.Hash]bool{}
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if !config.MatchAny(spec, ref.Name()) {
			if !ref.IsTag() || !wantTags {
				return nil
			}
		}

		if ref.Type() == plumbing.SymbolicReference {
			ref, err = storer.ResolveReference(remoteRefs, ref.Name())
			if err != nil {
				return err
			}
		}

		if ref.Type() != plumbing.HashReference {
			return nil
		}

		hash := ref.Hash()
		var exists bool

		if !ref.IsTag() {
			exists, err = objectExists(localStorer, hash)
			if err != nil {
				return err
			}
		} else {
			exists = hashInList(hash, tags)
		}

		if !exists {
			wants[hash] = true
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var result []plumbing.Hash
	for h := range wants {
		result = append(result, h)
	}

	return result, nil
}

func hashInList(h plumbing.Hash, list []plumbing.Hash) bool {
	for _, value := range list {
		if value == h {
			return true
		}
	}

	return false
}

func objectExists(s storer.EncodedObjectStorer, h plumbing.Hash) (bool, error) {
	_, err := s.EncodedObject(plumbing.AnyObject, h)
	if err == plumbing.ErrObjectNotFound {
		return false, nil
	}

	return true, err
}

func checkFastForwardUpdate(s storer.EncodedObjectStorer, remoteRefs storer.ReferenceStorer, cmd *packp.Command) error {
	if cmd.Old == plumbing.ZeroHash {
		_, err := remoteRefs.Reference(cmd.Name)
		if err == plumbing.ErrReferenceNotFound {
			return nil
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("non-fast-forward update: %s", cmd.Name.String())
	}

	ff, err := isFastForward(s, cmd.Old, cmd.New)
	if err != nil {
		return err
	}

	if !ff {
		return fmt.Errorf("non-fast-forward update: %s", cmd.Name.String())
	}

	return nil
}

func isFastForward(s storer.EncodedObjectStorer, old, new plumbing.Hash) (bool, error) {
	c, err := object.GetCommit(s, new)
	if err != nil {
		return false, err
	}

	found := false
	iter := object.NewCommitPreIterator(c)
	return found, iter.ForEach(func(c *object.Commit) error {
		if c.Hash != old {
			return nil
		}

		found = true
		return storer.ErrStop
	})
}

func (r *Remote) newUploadPackRequest(o *FetchOptions,
	ar *packp.AdvRefs) (*packp.UploadPackRequest, error) {

	req := packp.NewUploadPackRequestFromCapabilities(ar.Capabilities)

	if o.Depth != 0 {
		req.Depth = packp.DepthCommits(o.Depth)
		if err := req.Capabilities.Set(capability.Shallow); err != nil {
			return nil, err
		}
	}

	if o.Progress == nil && ar.Capabilities.Supports(capability.NoProgress) {
		if err := req.Capabilities.Set(capability.NoProgress); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func buildSidebandIfSupported(l *capability.List, reader io.Reader, p sideband.Progress) io.Reader {
	var t sideband.Type

	switch {
	case l.Supports(capability.Sideband):
		t = sideband.Sideband
	case l.Supports(capability.Sideband64k):
		t = sideband.Sideband64k
	default:
		return reader
	}

	d := sideband.NewDemuxer(t, reader)
	d.Progress = p

	return d
}

func (r *Remote) updateLocalReferenceStorage(specs []config.RefSpec, refs memory.ReferenceStorage) error {
	for _, spec := range specs {
		for _, ref := range refs {
			if !spec.Match(ref.Name()) {
				continue
			}

			if ref.Type() != plumbing.HashReference {
				continue
			}

			name := spec.Dst(ref.Name())
			n := plumbing.NewHashReference(name, ref.Hash())
			if err := r.s.SetReference(n); err != nil {
				return err
			}
		}
	}

	return r.buildFetchedTags(refs)
}

func (r *Remote) buildFetchedTags(refs storer.ReferenceStorer) error {
	iter, err := refs.IterReferences()
	if err != nil {
		return err
	}

	return iter.ForEach(func(ref *plumbing.Reference) error {
		if !ref.IsTag() {
			return nil
		}

		_, err := r.s.EncodedObject(plumbing.AnyObject, ref.Hash())
		if err == plumbing.ErrObjectNotFound {
			return nil
		}

		if err != nil {
			return err
		}

		return r.s.SetReference(ref)
	})
}

func objectsToPush(commands []*packp.Command) ([]plumbing.Hash, error) {
	var objects []plumbing.Hash
	for _, cmd := range commands {
		if cmd.New == plumbing.ZeroHash {
			continue
		}

		objects = append(objects, cmd.New)
	}

	return objects, nil
}

func referencesToHashes(refs storer.ReferenceStorer) ([]plumbing.Hash, error) {
	iter, err := refs.IterReferences()
	if err != nil {
		return nil, err
	}

	var hs []plumbing.Hash
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() != plumbing.HashReference {
			return nil
		}

		hs = append(hs, ref.Hash())
		return nil
	})
	if err != nil {
		return nil, err
	}

	return hs, nil
}

func pushHashes(sess transport.ReceivePackSession, sto storer.EncodedObjectStorer,
	req *packp.ReferenceUpdateRequest, hs []plumbing.Hash) (*packp.ReportStatus, error) {

	rd, wr := io.Pipe()
	req.Packfile = rd
	done := make(chan error)
	go func() {
		e := packfile.NewEncoder(wr, sto, false)
		if _, err := e.Encode(hs); err != nil {
			done <- wr.CloseWithError(err)
			return
		}

		done <- wr.Close()
	}()

	rs, err := sess.ReceivePack(req)
	if err != nil {
		return nil, err
	}

	if err := <-done; err != nil {
		return nil, err
	}

	return rs, nil
}

func (r *Remote) updateShallow(o *FetchOptions, resp *packp.UploadPackResponse) error {
	if o.Depth == 0 {
		return nil
	}

	return r.s.SetShallow(resp.Shallows)
}
