package ssh_test

import (
	"io/ioutil"
	"testing"

	"gopkg.in/src-d/go-git.v2/clients/common"
	. "gopkg.in/src-d/go-git.v2/clients/ssh"
)

const fixtureRepo = "git@github.com:tyba/git-fixture.git"

func TestConnect(t *testing.T) {
	s := NewGitUploadPackService()
	if err := s.Connect(fixtureRepo); err != nil {
		t.Error("cannot connect:", err)
	}
	defer s.Disconnect()
}

func TestInfo(t *testing.T) {
	s := NewGitUploadPackService()
	if err := s.Connect(fixtureRepo); err != nil {
		t.Fatal("cannot connect:", err)
	}
	defer s.Disconnect()

	i, err := s.Info()
	if err != nil {
		t.Error(err)
	} else if i == nil {
		t.Error("nil info")
	}
}

func TestDefaultBranch(t *testing.T) {
	s := NewGitUploadPackService()
	if err := s.Connect(fixtureRepo); err != nil {
		t.Fatal("cannot connect:", err)
	}
	defer s.Disconnect()

	i, err := s.Info()
	if err != nil {
		t.Fatal("cannot get info:", err)
	} else if i == nil {
		t.Fatal("nil info")
	}

	expHead := "refs/heads/master"
	head := i.Capabilities.SymbolicReference("HEAD")
	if head != expHead {
		t.Errorf("wrong head\n\texpected head = %s\n\treceived head = %s\n", expHead, head)
	}
}

func TestCapabilities(t *testing.T) {
	s := NewGitUploadPackService()
	if err := s.Connect(fixtureRepo); err != nil {
		t.Fatal("cannot connect:", err)
	}
	defer s.Disconnect()

	i, err := s.Info()
	if err != nil {
		t.Fatal("cannot get info:", err)
	} else if i == nil {
		t.Fatal("nil info")
	}

	expLen := 1
	length := len(i.Capabilities.Get("agent"))
	if expLen != length {
		t.Errorf("wrong length:\n\texpected = %d\n\tfound = %d\n", expLen, length)
	}
}

func TestFetch(t *testing.T) {
	s := NewGitUploadPackService()
	if err := s.Connect(fixtureRepo); err == nil {
		t.Fatal("cannot connect:", err)
	}
	defer s.Disconnect()

	r, err := s.Fetch(&common.GitUploadPackRequest{
		Want: []string{"6ecf0ef2c2dffb796033e5a02219af86ec6584e5"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if r == nil {
		t.Fatal("nil fetch")
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal("cannot ReadAll from fetch")
	}
	expLen := 85374
	length := len(b)
	if expLen != length {
		t.Errorf("wrong length:\n\texpected = %d\n\tfound = %d\n", expLen, length)
	}
}
