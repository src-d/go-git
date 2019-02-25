package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/_examples/merge_base/repository"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	. "gopkg.in/src-d/go-git.v4/_examples"
)

func helpAndExit(s string, exitCode int) {
	if s != "" {
		Warning("%s", s)
	}

	Warning("%s %s", os.Args[0], "<path> <baseRev> <headRev>")
	Warning("%s %s", os.Args[0], "<path> --is-ancestor <baseRev> <headRev>")
	Warning("%s %s", os.Args[0], "<path> --independent <commitRev>...")

	os.Exit(exitCode)
}

// Command that mimics `git merge-base <baseRev> <headRev>`
// Command that mimics `git merge-base --is-ancestor <baseRev> <headRev>`
// Command that mimics `git merge-base --independent <commitRev>...`
func main() {

	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		helpAndExit("Returns the merge-base between two commits:", 0)
	}

	if len(os.Args) < 4 {
		helpAndExit("Wrong syntax, Usage:", 1)
	}

	path := os.Args[1]

	var modeIndependent, modeAncestor bool
	var commitRevs []string
	var res []*object.Commit

	switch os.Args[2] {
	case "--independent":
		modeIndependent = true
		commitRevs = os.Args[3:]
	case "--is-ancestor":
		modeAncestor = true
		commitRevs = os.Args[3:]
		if len(os.Args) != 5 {
			helpAndExit("Wrong syntax, Usage:", 1)
		}
	default:
		commitRevs = os.Args[2:]
		if len(os.Args) != 4 {
			helpAndExit("Wrong syntax, Usage:", 1)
		}
	}

	// Open a git repository from the given path
	repo, err := git.PlainOpen(path)
	CheckIfError(err)

	// Get the hashes of the passed revisions
	var hashes []*plumbing.Hash
	for _, rev := range commitRevs {
		hash, err := repo.ResolveRevision(plumbing.Revision(rev))
		CheckIfError(wrappErr(err, "could not parse revision '%s'", rev))
		hashes = append(hashes, hash)
	}

	// Get the commits identified by the passed hashes
	var commits []*object.Commit
	for _, hash := range hashes {
		commit, err := repo.CommitObject(*hash)
		CheckIfError(wrappErr(err, "could not find commit '%s'", hash.String()))
		commits = append(commits, commit)
	}

	if modeAncestor {
		isAncestor, err := repository.IsAncestor(commits[0], commits[1])
		CheckIfError(err)

		if !isAncestor {
			Warning("%s is not ancestor of %s", commitRevs[0], commitRevs[1])
			os.Exit(1)
		}

		os.Exit(0)
	}

	if modeIndependent {
		res, err = repository.Independents(commits)
		CheckIfError(err)
	} else {
		// REVIEWER: store param wouldn't be needed if MergeBase were part of git.Repository
		res, err = repository.MergeBase(repo.Storer, commits[0], commits[1])
		CheckIfError(err)

		if len(res) == 0 {
			os.Exit(1)
		}
	}

	for _, commit := range res {
		print(commit)
	}
}

func wrappErr(err error, s string, v ...interface{}) error {
	if err != nil {
		return fmt.Errorf("%s: %s", fmt.Sprintf(s, v...), err)
	}

	return nil
}

func print(c *object.Commit) {
	if os.Getenv("LOG_LEVEL") == "verbose" {
		fmt.Printf(
			"\x1b[36;1m%s \x1b[90;21m%s\x1b[0m %s\n",
			c.Hash.String()[:7],
			c.Hash.String(),
			strings.Split(c.Message, "\n")[0],
		)
	} else {
		fmt.Println(c.Hash.String())
	}
}
