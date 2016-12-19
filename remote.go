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
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

var NoErrAlreadyUpToDate = errors.New("already up-to-date")

// Remote represents a connection to a remote repository
type Remote struct {
	c *config.RemoteConfig
	s Storer
	p sideband.Progress
}

func newRemote(s Storer, p sideband.Progress, c *config.RemoteConfig) *Remote {
	return &Remote{s: s, p: p, c: c}
}

// Config return the config
func (r *Remote) Config() *config.RemoteConfig {
	return r.c
}

func (r *Remote) String() string {
	fetch := r.c.URL
	push := r.c.URL

	return fmt.Sprintf("%s\t%s (fetch)\n%[1]s\t%s (push)", r.c.Name, fetch, push)
}

// Fetch fetches references from the remote to the local repository.
func (r *Remote) Fetch(o *FetchOptions) error {
	_, err := r.fetch(o)
	return err
}

// Push performs a push to the remote. Returns NoErrAlreadyUpToDate if the
// remote was already up-to-date.
//
// TODO: Support deletes.
// TODO: Support pushing tags.
// TODO: Check if force update is given, otherwise reject non-fast forward.
func (r *Remote) Push(o *PushOptions) (err error) {
	if o.RemoteName == "" {
		o.RemoteName = r.c.Name
	}

	if err := o.Validate(); err != nil {
		return err
	}

	if o.RemoteName != r.c.Name {
		return fmt.Errorf("remote names don't match: %s != %s", o.RemoteName, r.c.Name)
	}

	s, err := newSendPackSession(r.c.URL)
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

	commits, err := commitsToPush(r.s, req.Commands)
	if err != nil {
		return err
	}

	haves, err := referencesToHashes(remoteRefs)
	if err != nil {
		return err
	}

	hashesToPush, err := revlist.Objects(r.s, commits, haves)
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

	s, err := newFetchPackSession(r.c.URL)
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

func newFetchPackSession(url string) (transport.FetchPackSession, error) {
	c, ep, err := newClient(url)
	if err != nil {
		return nil, err
	}

	return c.NewFetchPackSession(ep)
}

func newSendPackSession(url string) (transport.SendPackSession, error) {
	c, ep, err := newClient(url)
	if err != nil {
		return nil, err
	}

	return c.NewSendPackSession(ep)
}

func newClient(url string) (transport.Client, transport.Endpoint, error) {
	ep, err := transport.NewEndpoint(url)
	if err != nil {
		return nil, transport.Endpoint{}, err
	}

	c, err := client.NewClient(ep)
	if err != nil {
		return nil, transport.Endpoint{}, err
	}

	return c, ep, err
}

func (r *Remote) fetchPack(o *FetchOptions, s transport.FetchPackSession,
	req *packp.UploadPackRequest) (err error) {

	reader, err := s.FetchPack(req)
	if err != nil {
		return err
	}

	defer ioutil.CheckClose(reader, &err)

	if err := r.updateShallow(o, reader); err != nil {
		return err
	}

	if err = r.updateObjectStorage(
		buildSidebandIfSupported(req.Capabilities, reader, r.p),
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

	dstName := rs.Dst(localRef.Name())
	oldHash := plumbing.ZeroHash
	newHash := localRef.Hash()

	iter, err := remoteRefs.IterReferences()
	if err != nil {
		return err
	}

	err = iter.ForEach(func(remoteRef *plumbing.Reference) error {
		if remoteRef.Type() != plumbing.HashReference {
			return nil
		}

		if dstName != remoteRef.Name() {
			return nil
		}

		oldHash = remoteRef.Hash()
		return nil
	})

	if oldHash == newHash {
		return nil
	}

	req.Commands = append(req.Commands, &packp.Command{
		Name: dstName,
		Old:  oldHash,
		New:  newHash,
	})
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

func getWants(spec []config.RefSpec, localStorer Storer, remoteRefs storer.ReferenceStorer) ([]plumbing.Hash, error) {
	wantTags := true
	for _, s := range spec {
		if !s.IsWildcard() {
			wantTags = false
			break
		}
	}

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
		exists, err := commitExists(localStorer, hash)
		if err != nil {
			return err
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

func commitExists(s storer.EncodedObjectStorer, h plumbing.Hash) (bool, error) {
	_, err := s.EncodedObject(plumbing.CommitObject, h)
	if err == plumbing.ErrObjectNotFound {
		return false, nil
	}

	return true, err
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

	if r.p == nil && ar.Capabilities.Supports(capability.NoProgress) {
		if err := req.Capabilities.Set(capability.NoProgress); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func (r *Remote) updateObjectStorage(reader io.Reader) error {
	if sw, ok := r.s.(storer.PackfileWriter); ok {
		w, err := sw.PackfileWriter()
		if err != nil {
			return err
		}

		defer w.Close()
		_, err = io.Copy(w, reader)
		return err
	}

	stream := packfile.NewScanner(reader)
	d, err := packfile.NewDecoder(stream, r.s)
	if err != nil {
		return err
	}

	_, err = d.Decode()
	return err
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

func commitsToPush(s storer.EncodedObjectStorer, commands []*packp.Command) ([]*object.Commit, error) {
	var commits []*object.Commit
	for _, cmd := range commands {
		if cmd.New == plumbing.ZeroHash {
			continue
		}

		c, err := object.GetCommit(s, cmd.New)
		if err != nil {
			return nil, err
		}

		commits = append(commits, c)
	}

	return commits, nil
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

func pushHashes(sess transport.SendPackSession, sto storer.EncodedObjectStorer,
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

	rs, err := sess.SendPack(req)
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
