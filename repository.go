package git

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/src-d/go-git.v3/clients/common"
	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/file"
	"gopkg.in/src-d/go-git.v3/formats/packfile"
	"gopkg.in/src-d/go-git.v3/storage/memory"
	"gopkg.in/src-d/go-git.v3/storage/seekable"
)

var (
	// ErrObjectNotFound object not found
	ErrObjectNotFound = errors.New("object not found")
)

const (
	// DefaultRemoteName name of the default Remote, just like git command
	DefaultRemoteName = "origin"
)

// Repository git repository struct
type Repository struct {
	Remotes map[string]*Remote
	Storage core.ObjectStorage
	URL     string
}

// NewRepository creates a new repository setting remote as default remote
func NewRepository(url string, auth common.AuthMethod) (*Repository, error) {
	repo := NewPlainRepository()
	repo.URL = url

	if isRemote(url) {
		r, err := NewAuthenticatedRemote(url, auth)
		repo.Remotes[DefaultRemoteName] = r
		if err != nil {
			return nil, err
		}
	} else {
		repo.useLocalStorage(url)
	}

	return repo, nil
}

const fileScheme = "file://"

func isRemote(url string) bool {
	return !strings.HasPrefix(url, fileScheme)
}

func (r *Repository) useLocalStorage(url string) error {
	path := strings.TrimPrefix(url, fileScheme)
	dir, err := file.NewDir(path)
	if err != nil {
		return err
	}

	packfile, err := dir.Packfile()
	if err != nil {
		return err
	}

	idxfile, err := dir.Idxfile()
	if err != nil {
		// if there is no idx file, just keep on, we will manage to create one
		// on the fly.
		if err != file.ErrIdxNotFound {
			return err
		}
	}

	r.Storage, err = seekable.New(packfile, idxfile)

	return err
}

// NewPlainRepository creates a new repository without remotes
func NewPlainRepository() *Repository {
	return &Repository{
		Remotes: map[string]*Remote{},
		Storage: memory.NewObjectStorage(),
	}
}

// Pull connect and fetch the given branch from the given remote, the branch
// should be provided with the full path not only the abbreviation, eg.:
// "refs/heads/master"
func (r *Repository) Pull(remoteName, branch string) (err error) {
	remote, ok := r.Remotes[remoteName]
	if !ok {
		return fmt.Errorf("unable to find remote %q", remoteName)
	}

	if err := remote.Connect(); err != nil {
		return err
	}

	if branch == "" {
		branch = remote.DefaultBranch()
	}

	ref, err := remote.Ref(branch)
	if err != nil {
		return err
	}

	req := &common.GitUploadPackRequest{}
	req.Want(ref)

	// TODO: Provide "haves" for what's already in the repository's storage

	reader, err := remote.Fetch(req)
	if err != nil {
		return err
	}
	defer checkClose(reader, &err)

	d := packfile.NewDecoder(reader)
	if _, err = d.Decode(r.Storage); err != nil {
		return err
	}

	return nil
}

// PullDefault like Pull but retrieve the default branch from the default remote
func (r *Repository) PullDefault() (err error) {
	return r.Pull(DefaultRemoteName, "")
}

// Commit return the commit with the given hash
func (r *Repository) Commit(h core.Hash) (*Commit, error) {
	obj, err := r.Storage.Get(h)
	if err != nil {
		if err == core.ErrObjectNotFound {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	commit := &Commit{r: r}
	return commit, commit.Decode(obj)
}

// Commits decode the objects into commits
func (r *Repository) Commits() (*CommitIter, error) {
	iter, err := r.Storage.Iter(core.CommitObject)
	if err != nil {
		return nil, err
	}

	return NewCommitIter(r, iter), nil
}

// Tree return the tree with the given hash
func (r *Repository) Tree(h core.Hash) (*Tree, error) {
	obj, err := r.Storage.Get(h)
	if err != nil {
		if err == core.ErrObjectNotFound {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	tree := &Tree{r: r}
	return tree, tree.Decode(obj)
}

// Blob returns the blob with the given hash
func (r *Repository) Blob(h core.Hash) (*Blob, error) {
	obj, err := r.Storage.Get(h)
	if err != nil {
		if err == core.ErrObjectNotFound {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	blob := &Blob{}
	return blob, blob.Decode(obj)
}

// Tag returns a tag with the given hash.
func (r *Repository) Tag(h core.Hash) (*Tag, error) {
	obj, err := r.Storage.Get(h)
	if err != nil {
		if err == core.ErrObjectNotFound {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	t := &Tag{r: r}
	return t, t.Decode(obj)
}

// Tags returns a TagIter that can step through all of the annotated tags
// in the repository.
func (r *Repository) Tags() (*TagIter, error) {
	iter, err := r.Storage.Iter(core.TagObject)
	if err != nil {
		return nil, err
	}

	return NewTagIter(r, iter), nil
}

// Object returns an object with the given hash.
func (r *Repository) Object(h core.Hash) (Object, error) {
	obj, err := r.Storage.Get(h)
	if err != nil {
		if err == core.ErrObjectNotFound {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	switch obj.Type() {
	case core.CommitObject:
		commit := &Commit{r: r}
		return commit, commit.Decode(obj)
	case core.TreeObject:
		tree := &Tree{r: r}
		return tree, tree.Decode(obj)
	case core.BlobObject:
		blob := &Blob{}
		return blob, blob.Decode(obj)
	case core.TagObject:
		tag := &Tag{r: r}
		return tag, tag.Decode(obj)
	default:
		return nil, core.ErrInvalidType
	}
}
