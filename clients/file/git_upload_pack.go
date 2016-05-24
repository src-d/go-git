package file

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/src-d/go-git.v3/clients/common"
	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/gitdir"
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

	fmt.Println(info.Refs)

	info.Head = core.ZeroHash
	return info, nil
}

func (s *GitUploadPackService) Fetch(r *common.GitUploadPackRequest) (io.ReadCloser, error) {
	return nil, nil
}
