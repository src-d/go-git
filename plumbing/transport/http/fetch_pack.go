package http

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packp/pktline"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

// FetchPackInfo returns the references info and capabilities from the service.
func (c *Client) FetchPackInfo() (*transport.UploadPackInfo, error) {
	url := fmt.Sprintf(
		"%s/info/refs?service=%s",
		c.ep.String(), transport.UploadPackServiceName,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	c.applyAuthToRequest(req)
	c.applyHeadersToRequest(req, nil)
	res, err := c.c.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized {
		return nil, transport.ErrAuthorizationRequired
	}

	i := transport.NewUploadPackInfo()
	return i, i.Decode(res.Body)
}

// Fetch request and returns a reader to a packfile
func (c *Client) FetchPack(r *transport.UploadPackRequest) (io.ReadCloser, error) {
	url := fmt.Sprintf(
		"%s/%s",
		c.ep.String(), transport.UploadPackServiceName,
	)

	res, err := c.doRequest("POST", url, r.Reader())
	if err != nil {
		return nil, err
	}

	reader := newBufferedReadCloser(res.Body)
	if _, err := reader.Peek(1); err != nil {
		if err == io.ErrUnexpectedEOF {
			return nil, transport.ErrEmptyUploadPackRequest
		}

		return nil, err
	}

	if err := discardResponseInfo(reader); err != nil {
		return nil, err
	}

	return reader, nil
}

func discardResponseInfo(r io.Reader) error {
	s := pktline.NewScanner(r)
	for s.Scan() {
		if bytes.Equal(s.Bytes(), []byte{'N', 'A', 'K', '\n'}) {
			break
		}
	}

	return s.Err()
}

func (s *Client) doRequest(method, url string, content *strings.Reader) (*http.Response, error) {
	var body io.Reader
	if content != nil {
		body = content
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, plumbing.NewPermanentError(err)
	}

	s.applyHeadersToRequest(req, content)
	s.applyAuthToRequest(req)

	res, err := s.c.Do(req)
	if err != nil {
		return nil, plumbing.NewUnexpectedError(err)
	}

	if err := NewErr(res); err != nil {
		_ = res.Body.Close()
		return nil, err
	}

	return res, nil
}

func (c *Client) applyHeadersToRequest(req *http.Request, content *strings.Reader) {
	req.Header.Add("User-Agent", "git/1.0")
	req.Header.Add("Host", "github.com")

	if content == nil {
		req.Header.Add("Accept", "*/*")
	} else {
		req.Header.Add("Accept", "application/x-git-upload-pack-result")
		req.Header.Add("Content-Type", "application/x-git-upload-pack-request")
		req.Header.Add("Content-Length", string(content.Len()))
	}
}

func (c *Client) applyAuthToRequest(req *http.Request) {
	if c.auth == nil {
		return
	}

	c.auth.setAuth(req)
}

type bufferedReadCloser struct {
	*bufio.Reader
	closer io.Closer
}

func newBufferedReadCloser(r io.ReadCloser) *bufferedReadCloser {
	return &bufferedReadCloser{bufio.NewReader(r), r}
}

func (r *bufferedReadCloser) Close() error {
	return r.closer.Close()
}
