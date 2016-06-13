package file

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"gopkg.in/src-d/go-git.v3/clients/common"
	"gopkg.in/src-d/go-git.v3/formats/file"
)

var (
	ErrHeadSymRefNotFound = errors.New("HEAD symbolic reference not found")
)

type GitUploadPackService struct {
	dir *file.Dir
}

func NewGitUploadPackService() *GitUploadPackService {
	return &GitUploadPackService{}
}

func (s *GitUploadPackService) Connect(url common.Endpoint) error {
	var err error

	p := strings.TrimPrefix(string(url), "file://")
	s.dir, err = file.NewDir(p)
	if err != nil {
		return err
	}

	return nil
}

func (s *GitUploadPackService) ConnectWithAuth(url common.Endpoint,
	a common.AuthMethod) error {

	if a == nil {
		return s.Connect(url)
	}

	return common.ErrAuthNotSupported
}

func (s *GitUploadPackService) Info() (*common.GitUploadPackInfo, error) {
	i := common.NewGitUploadPackInfo()
	var err error

	if i.Refs, err = s.dir.Refs(); err != nil {
		return i, err
	}

	if i.Capabilities, err = s.dir.Capabilities(); err != nil {
		return i, err
	}

	h := i.Capabilities.SymbolicReference("HEAD")
	var ok bool
	if i.Head, ok = i.Refs[h]; !ok {
		return i, ErrHeadSymRefNotFound
	}

	return i, nil
}

func (s *GitUploadPackService) Fetch(r *common.GitUploadPackRequest) (io.ReadCloser, error) {
	return nil, fmt.Errorf("fetch makes no sense for dir clients")
}
