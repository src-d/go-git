package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
)

const (
	bin            = "go-git"
	receivePackBin = "git-receive-pack"
	uploadPackBin  = "git-upload-pack"
)

func main() {
	switch filepath.Base(os.Args[0]) {
	case receivePackBin:
		os.Args = append([]string{"git", "receive-pack"}, os.Args[1:]...)
	case uploadPackBin:
		os.Args = append([]string{"git", "upload-pack"}, os.Args[1:]...)
	}

	parser := flags.NewNamedParser(bin, flags.Default)
	parser.AddCommand("receive-pack", "", "", &CmdReceivePack{})
	parser.AddCommand("upload-pack", "", "", &CmdUploadPack{})
	parser.AddCommand("version", "Show the version information.", "", &CmdVersion{})

	_, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrCommandRequired {
			parser.WriteHelp(os.Stdout)
		}

		os.Exit(1)
	}
}

type cmd struct {
	Verbose bool `short:"v" description:"Activates the verbose mode"`
}

func (c *cmd) print(format string, a ...interface{}) {
	if !c.Verbose {
		return
	}

	fmt.Printf(format, a...)
}

func resolvePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return path, err
	}

	return filepath.Join(wd, path), nil
}
