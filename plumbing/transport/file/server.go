package file

import (
	"fmt"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/server"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"

	osfs "srcd.works/go-billy.v1/os"
)

var DefaultServer = NewServer(server.DefaultHandler)

type Server struct {
	handler server.Handler
}

func NewServer(handler server.Handler) *Server {
	return &Server{handler}
}

func (s *Server) Serve(cmd string, args []string) error {
	switch cmd {
	case transport.UploadPackServiceName:
		return serveUploadPackServiceName(s.handler, args)
	case transport.ReceivePackServiceName:
		return serveReceivePackServiceName(s.handler, args)
	default:
		return fmt.Errorf("invalid command: %s", cmd)
	}
}

var srvCmd = server.ServerCommand{
	Stdin:  os.Stdin,
	Stdout: ioutil.WriteNopCloser(os.Stdout),
	Stderr: os.Stderr,
}

func serveUploadPackServiceName(h server.Handler, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("only one argument is currently supported")
	}

	path := args[0]
	s, err := newStorerByPath(path)
	if err != nil {
		return fmt.Errorf("error creating storer: %s", err)
	}

	sess, err := h.NewUploadPackSession(s)
	if err != nil {
		return fmt.Errorf("error creating session: %s", err)
	}

	return server.UploadPack(srvCmd, sess)
}

func serveReceivePackServiceName(h server.Handler, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("only one argument is currently supported")
	}

	path := args[0]
	s, err := newStorerByPath(path)
	if err != nil {
		return fmt.Errorf("error creating storer: %s", err)
	}

	sess, err := h.NewReceivePackSession(s)
	if err != nil {
		return fmt.Errorf("error creating session: %s", err)
	}

	return server.ReceivePack(srvCmd, sess)
}

func newStorerByPath(path string) (server.Storer, error) {
	return filesystem.NewStorage(osfs.New(path))
}
