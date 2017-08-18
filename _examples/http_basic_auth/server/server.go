package server

import (
	"github.com/phayes/freeport"
	"strconv"
	"github.com/AaronO/go-git-http"
	"log"
	"net/http"
	"os"
	. "gopkg.in/src-d/go-git.v4/_examples"
	"github.com/AaronO/go-git-http/auth"
	"io/ioutil"
	"os/exec"
)

type HTTPBasicAuthServer struct {
	done                                   chan bool
	stopchan                               chan bool
	port                                   int
	logger                                 *log.Logger
	mux                                    *http.ServeMux
	rootdir, addr, URL, Username, Password string
}

var tempFolders = []string{}

func NewHTTPBasicAuthServer() *HTTPBasicAuthServer {
	port := freeport.GetPort()
	addr := "localhost:" + strconv.Itoa(port)
	url := "http://" + addr

	s := &HTTPBasicAuthServer{
		done:     make(chan bool, 1),
		stopchan: make(chan bool, 1),
		port:     port,
		logger:   log.New(os.Stdout, "", 0),
		mux:      http.NewServeMux(),
		rootdir:  tempFolder(),
		addr:     addr,
		URL:      url,
		Username: "admin",
		Password: "123456",
	}

	git := githttp.New(s.rootdir)

	// Build an authentication middleware based on a function
	authenticator := auth.Authenticator(func(info auth.AuthInfo) (bool, error) {
		// Disallow Pushes (making git server pull only)
		if info.Push {
			return false, nil
		}

		// Typically this would be a database lookup
		if info.Username == s.Username && info.Password == s.Password {
			return true, nil
		}

		return false, nil
	})

	s.mux.Handle("/", authenticator(git))

	return s
}

func (s *HTTPBasicAuthServer) run() *HTTPBasicAuthServer {
	h := &http.Server{Addr: s.addr, Handler: s}

	cloneBareRepository(DefaultURL, s.rootdir)

	go func() {
		log.Printf("Listening on %s\n", s.URL)

		if err := h.ListenAndServe(); err != nil {
			s.logger.Fatal(err)
		}
	}()

	return s
}

func (s *HTTPBasicAuthServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

var DefaultURL = "https://github.com/git-fixtures/basic.git"

func tempFolder() string {
	path, err := ioutil.TempDir("", "")
	CheckIfError(err)

	tempFolders = append(tempFolders, path)
	return path
}

func cloneBareRepository(url, folder string) string {
	cmd := exec.Command("git", "clone", "--bare", url, folder)
	err := cmd.Run()
	CheckIfError(err)

	return folder
}

func WithServer(f func(server *HTTPBasicAuthServer)) {
	defer deleteTempFolders()
	s := NewHTTPBasicAuthServer()
	s.run()
	f(s)
}


func deleteTempFolders() {
	for _, folder := range tempFolders {
		err := os.RemoveAll(folder)
		CheckIfError(err)
	}
}