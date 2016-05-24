package file

import (
	"fmt"
	"io"

	"gopkg.in/src-d/go-git.v3/clients/common"
)

type GitUploadPackService struct {
	path string
}

func NewGitUploadPackService() *GitUploadPackService {
	return &GitUploadPackService{}
}

func (s *GitUploadPackService) Connect(url common.Endpoint) error {
	fmt.Println(url)
	return nil
}

func (s *GitUploadPackService) ConnectWithAuth(url common.Endpoint, auth common.AuthMethod) error {
	if auth == nil {
		return s.Connect(url)
	}

	return common.ErrAuthNotSupported
}

func (s *GitUploadPackService) Info() (*common.GitUploadPackInfo, error) {
	return nil, nil
}

func (s *GitUploadPackService) Fetch(r *common.GitUploadPackRequest) (io.ReadCloser, error) {
	return nil, nil
}
