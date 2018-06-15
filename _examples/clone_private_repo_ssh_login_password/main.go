package main

import (
	"fmt"
	"os"

	"net"
	"bufio"
	"strings"
	"golang.org/x/crypto/ssh/terminal"

	"golang.org/x/crypto/ssh"
	gogitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4"
	. "gopkg.in/src-d/go-git.v4/_examples"
	
)

// Basic example of how to clone a private repository using ssh and user, password  authorisation

func GetCredential() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(0)
	if err != nil {
		panic(err)
	}
	password := string(bytePassword)

	fmt.Println("")
	
	return strings.TrimSpace(username), strings.TrimSpace(password)
}


func main() {
	CheckArgs("<url>", "<directory>")
	url := os.Args[1]
	directory := os.Args[2]

	username, password := GetCredential()
	
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}

	auth := &gogitssh.Password{User: username, Password: password, HostKeyCallbackHelper: gogitssh.HostKeyCallbackHelper{
		HostKeyCallback: hostKeyCallback,
	}}
	
	// Clone the given repository to the given directory
	Info("git clone %s %s --recursive", url, directory)

	r, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth: auth,
	})

	CheckIfError(err)

	// ... retrieving the branch being pointed by HEAD
	ref, err := r.Head()
	CheckIfError(err)
	// ... retrieving the commit object
	commit, err := r.CommitObject(ref.Hash())
	CheckIfError(err)

	fmt.Println(commit)
}
