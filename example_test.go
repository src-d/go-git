package git_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/src-d/go-git/plumbing/object"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"

	"gopkg.in/src-d/go-billy.v4/memfs"
)

// repository is a boilerplate sample repository (go-git itself).
var repository *git.Repository

func init() {
	cwd, err := os.Getwd()
	if err == nil {
		for true {
			files, err := ioutil.ReadDir(cwd)
			if err != nil {
				break
			}
			found := false
			for _, f := range files {
				if f.Name() == "README.md" {
					found = true
					break
				}
			}
			if found {
				break
			}
			oldCwd := cwd
			cwd = path.Dir(cwd)
			if oldCwd == cwd {
				break
			}
		}
		Repository, err = git.PlainOpen(cwd)
		if err == nil {
			iter, err := Repository.CommitObjects()
			if err == nil {
				commits := -1
				for ; err != io.EOF; _, err = iter.Next() {
					if err != nil {
						panic(err)
					}
					commits++
					if commits >= 100 {
						return
					}
				}
			}
		}
	}
	Repository, err = git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: "https://github.com/src-d/go-git",
	})
	if err != nil {
		panic(err)
	}
}

func ExampleClone() {
	// Filesystem abstraction based on memory
	fs := memfs.New()
	// Git objects storer based on memory
	storer := memory.NewStorage()

	// Clones the repository into the worktree (fs) and storer all the .git
	// content into the storer
	_, err := git.Clone(storer, fs, &git.CloneOptions{
		URL: "https://github.com/git-fixtures/basic.git",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Prints the content of the CHANGELOG file from the cloned repository
	changelog, err := fs.Open("CHANGELOG")
	if err != nil {
		log.Fatal(err)
	}

	io.Copy(os.Stdout, changelog)
	// Output: Initial changelog
}

func ExamplePlainClone() {
	// Tempdir to clone the repository
	dir, err := ioutil.TempDir("", "clone-example")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(dir) // clean up

	// Clones the repository into the given dir, just as a normal git clone does
	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL: "https://github.com/git-fixtures/basic.git",
	})

	if err != nil {
		log.Fatal(err)
	}

	// Prints the content of the CHANGELOG file from the cloned repository
	changelog, err := os.Open(filepath.Join(dir, "CHANGELOG"))
	if err != nil {
		log.Fatal(err)
	}

	io.Copy(os.Stdout, changelog)
	// Output: Initial changelog
}

func ExamplePlainClone_usernamePassword() {
	// Tempdir to clone the repository
	dir, err := ioutil.TempDir("", "clone-example")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(dir) // clean up

	// Clones the repository into the given dir, just as a normal git clone does
	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL: "https://github.com/git-fixtures/basic.git",
		Auth: &http.BasicAuth{
			Username: "username",
			Password: "password",
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}

func ExamplePlainClone_accessToken() {
	// Tempdir to clone the repository
	dir, err := ioutil.TempDir("", "clone-example")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(dir) // clean up

	// Clones the repository into the given dir, just as a normal git clone does
	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL: "https://github.com/git-fixtures/basic.git",
		Auth: &http.BasicAuth{
			Username: "abc123", // anything except an empty string
			Password: "github_access_token",
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}

func ExampleRepository_References() {
	r, _ := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: "https://github.com/git-fixtures/basic.git",
	})

	// simulating a git show-ref
	refs, _ := r.References()
	refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.HashReference {
			fmt.Println(ref)
		}

		return nil
	})

	// Example Output:
	// 6ecf0ef2c2dffb796033e5a02219af86ec6584e5 refs/remotes/origin/master
	// e8d3ffab552895c19b9fcf7aa264d277cde33881 refs/remotes/origin/branch
	// 6ecf0ef2c2dffb796033e5a02219af86ec6584e5 refs/heads/master

}

func ExampleRepository_CreateRemote() {
	r, _ := git.Init(memory.NewStorage(), nil)

	// Add a new remote, with the default fetch refspec
	_, err := r.CreateRemote(&config.RemoteConfig{
		Name: "example",
		URLs: []string{"https://github.com/git-fixtures/basic.git"},
	})

	if err != nil {
		log.Fatal(err)
	}

	list, err := r.Remotes()
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range list {
		fmt.Println(r)
	}

	// Example Output:
	// example https://github.com/git-fixtures/basic.git (fetch)
	// example https://github.com/git-fixtures/basic.git (push)
}

func ExampleDiffRepo(t *testing.T) {
	var hash plumbing.Hash

	prevHash := plumbing.NewHash("5fddbeb678bd2c36c5e5c891ab8f2b143ced5baf")
	hash := plumbing.NewHash("c088fd6a7e1a38e9d5a9815265cb575bb08d08ff")

	prevCommit, err := repo.CommitObject(prevHash)
	if err != nil {
		t.Fatalf("repo.CommitObject(%q) error: %v", prevHash, err)
	}

	commit, err := repo.CommitObject(hash)
	if err != nil {
		t.Fatalf("repo.CommitObject(%q) error: %v", hash, err)
	}

	fmt.Println("Comparing from:" + prevCommit.Hash.String() + " to:" + commit.Hash.String())

	isAncestor, err := commit.IsAncestor(prevCommit)
	if err != nil {
		t.Fatalf("commit.IsAncestor(%q) error: %v", prevCommit, err)
	}

	fmt.Printf("Is the prevCommit an ancestor of commit? : %v %v\n", isAncestor)

	currentTree, err := commit.Tree()
	if err != nil {
		t.Errorf("commit.Tree() error: %v", err)
	}

	prevTree, err := prevCommit.Tree()
	if err != nil {
		t.Errorf("prevCommit.Tree() error: %v", err)
	}

	patch, err := currentTree.Patch(prevTree)
	if err != nil {
		t.Errorf("currentTree.Patch(%q) error: %v", prevTree, err)
	}
	fmt.Println("Got here" + strconv.Itoa(len(patch.Stats())))

	var changedFiles []string
	for _, fileStat := range patch.Stats() {
		fmt.Println(fileStat.Name)
		changedFiles = append(changedFiles, fileStat.Name)
	}

	changes, err := currentTree.Diff(prevTree)
	if err != nil {
		t.Errorf("currentTree.Diff(%v) error: %v", prevTree, err)
	}

	fmt.Println("Got here!")
	for _, change := range changes {
		// Ignore deleted files
		action, err := change.Action()
		if err != nil {
			t.Errorf("change.Action() error: %v", err)
		}
		if action == merkletrie.Delete {
			fmt.Println("Skipping delete")
			continue
		}

		// Get list of involved files
		name := change.To.Name

		var empty = object.ChangeEntry{}
		if change.From != empty {
			name = change.From.Name
		}
		fmt.Println(name)
	}
}
