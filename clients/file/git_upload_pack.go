package file

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"gopkg.in/src-d/go-git.v3/clients/common"
	"gopkg.in/src-d/go-git.v3/formats/gitdir"
)

var (
	ErrHeadSymRefNotFound = errors.New("HEAD symbolic reference not found")
)

type GitUploadPackService struct {
	dir *gitdir.Dir
}

func NewGitUploadPackService() *GitUploadPackService {
	return &GitUploadPackService{}
}

func (s *GitUploadPackService) Connect(url common.Endpoint) error {
	var err error

	path := strings.TrimPrefix(string(url), "file://")
	s.dir, err = gitdir.New(path)
	if err != nil {
		return err
	}

	return nil
}

func (s *GitUploadPackService) ConnectWithAuth(url common.Endpoint,
	auth common.AuthMethod) error {

	if auth == nil {
		return s.Connect(url)
	}

	return common.ErrAuthNotSupported
}

func (s *GitUploadPackService) Info() (*common.GitUploadPackInfo, error) {
	info := common.NewGitUploadPackInfo()
	var err error

	if info.Refs, err = s.dir.Refs(); err != nil {
		return info, err
	}

	if info.Capabilities, err = s.dir.Capabilities(); err != nil {
		return info, err
	}

	headSymRef := info.Capabilities.SymbolicReference("HEAD")
	var ok bool
	if info.Head, ok = info.Refs[headSymRef]; !ok {
		return info, ErrHeadSymRefNotFound
	}

	return info, nil
}

func (s *GitUploadPackService) Fetch(r *common.GitUploadPackRequest) (io.ReadCloser, error) {
	return nil, fmt.Errorf("fetch makes no sense for dir clients")
}
