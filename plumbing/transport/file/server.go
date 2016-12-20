package file

import (
	"fmt"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/internal/common"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/server"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

var DefaultServer = NewServer(server.DefaultLoader, server.DefaultHandler)

type Server struct {
	loader  server.Loader
	handler server.Handler
}

func NewServer(loader server.Loader, handler server.Handler) *Server {
	return &Server{loader, handler}
}

func (s *Server) Serve(cmd string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("only one argument is currently supported")
	}

	sto, err := s.loader.Load("", args[0])
	if err != nil {
		return err
	}

	switch cmd {
	case transport.UploadPackServiceName:
		sess, err := s.handler.NewUploadPackSession(sto)
		if err != nil {
			return fmt.Errorf("error creating session: %s", err)
		}

		return common.UploadPack(srvCmd, sess)
	case transport.ReceivePackServiceName:
		sess, err := s.handler.NewReceivePackSession(sto)
		if err != nil {
			return fmt.Errorf("error creating session: %s", err)
		}

		return common.ReceivePack(srvCmd, sess)
	default:
		return fmt.Errorf("invalid command: %s", cmd)
	}
}

var srvCmd = common.ServerCommand{
	Stdin:  os.Stdin,
	Stdout: ioutil.WriteNopCloser(os.Stdout),
	Stderr: os.Stderr,
}
