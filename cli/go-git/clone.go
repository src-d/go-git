package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

type CmdClone struct {
	cmd

	Bare              []bool `long:"bare" description:"Make a bare Git repository."`
	Branch            string `long:"branch" value-name:"name" description:"Instead of pointing the newly created HEAD to the branch pointed to by the cloned repository’s HEAD, point to <name> branch instead."`
	Depth             int    `long:"depth" value-name:"depth" description:"Create a shallow clone with a history truncated to the specified number of commits."`
	Origin            string `long:"origin" short:"o" value-name:"name" description:"Instead of using the remote name origin to keep track of the upstream repository, use <name>." default:"origin"`
	Quiet             []bool `long:"quiet" short:"q" description:"Operate quietly. Progress is not reported to the standard error stream."`
	RecurseSubmodules []bool `long:"recurse-submodules" description:"After the clone is created, initialize all submodules within, using their default settings."`
	SingleBranch      []bool `long:"single-branch" description:"Clone only the history leading to the tip of a single branch, either specified by the --branch option or the primary branch remote’s HEAD points at."`
	Args              struct {
		Repository string `positional-arg-name:"repository" required:"true"`
		Directory  string `positional-arg-name:"directory"`
	} `positional-args:"yes"`
}

func (c *CmdClone) Execute(args []string) error {
	ep, err := transport.NewEndpoint(c.Args.Repository)
	if err != nil {
		return err
	}

	dir := c.Args.Directory
	if dir == "" {
		if dir, err = humanish(ep); err != nil {
			return err
		}
	}

	fmt.Printf("Cloning into '%s'...\n", dir)

	auth, err := initialAuth(ep)
	if err != nil {
		return err
	}

	opts := &git.CloneOptions{
		URL:          c.Args.Repository,
		Auth:         auth,
		RemoteName:   c.Origin,
		SingleBranch: optIsTrue(c.SingleBranch),
		Depth:        c.Depth,
	}

	if c.Branch != "" {
		opts.ReferenceName = plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", c.Branch))
	}

	if optIsTrue(c.RecurseSubmodules) {
		opts.RecurseSubmodules = git.DefaultSubmoduleRecursionDepth
	}

	if !optIsTrue(c.Quiet) {
		// TODO: Prefix 'remote: ' to every line.
		opts.Progress = os.Stderr
	}

	_, err = git.PlainClone(dir, optIsTrue(c.Bare), opts)
	if err == git.ErrRepositoryAlreadyExists {
		return err
	}

	if err != nil {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}

	if err == transport.ErrAuthenticationRequired {
		opts.Auth, err = retryAuth(ep)
		if err != nil {
			return err
		}

		_, err = git.PlainClone(dir, optIsTrue(c.Bare), opts)
	}

	if err != nil {
		_ = os.RemoveAll(dir)
		return err
	}

	return nil
}

func humanish(ep transport.Endpoint) (string, error) {
	p := path.Base(ep.Path())
	p = strings.TrimSuffix(p, ".git")
	if p == "." || p == "" {
		return "", fmt.Errorf("no valid repository name")
	}

	return p, nil
}

func initialAuth(ep transport.Endpoint) (transport.AuthMethod, error) {
	switch ep.Protocol() {
	case "ssh":
		auth, err := ssh.NewSSHAgentAuth(ep.User())
		if err != nil {
			return nil, err
		}

		return auth, nil
	default:
		return nil, nil
	}
}

func retryAuth(ep transport.Endpoint) (transport.AuthMethod, error) {
	switch ep.Protocol() {
	case "ssh":
		return nil, fmt.Errorf("asking for ssh password not supported yet")
	case "http", "https":
		var u, p string
		//TODO: read netrc first
		fmt.Printf("Username for '%s://%s': ", ep.Protocol(), ep.Host())
		if _, err := fmt.Scanln(&u); err != nil {
			return nil, err
		}

		fmt.Printf("Password for '%s://%s@%s': ", ep.Protocol(), u, ep.Host())
		if _, err := fmt.Scanln(&p); err != nil {
			return nil, err
		}

		return http.NewBasicAuth(u, p), nil

	}
	return nil, nil
}
