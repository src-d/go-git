package tgz

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

const (
	useDefaultTempDir = ""
	tmpPrefix         = "tmp-tgz-"
)

// Extract decompress a gziped tarball into a new temporal directory
// created just for this purpose.
//
// On success, the path of new directory and a nil error is returned.
//
// On error, a non-nil error and an empty string are returned if the
// newly created directory was correctly deleted. If not, its path is
// returned instead of the empty string.
func Extract(tgz string) (dir string, err error) {
	file, err := os.Open(tgz)
	if err != nil {
		return "", err
	}

	defer func() {
		errClose := file.Close()
		if err == nil {
			err = errClose
		}
	}()

	dir, err = ioutil.TempDir(useDefaultTempDir, tmpPrefix)
	if err != nil {
		return "", nil
	}

	tarReader, err := zipTarReader(file)
	if err != nil {
		return deleteDir(dir, err)
	}

	if err = unTar(tarReader, dir); err != nil {
		return deleteDir(dir, err)
	}

	return dir, nil
}

func deleteDir(dir string, prevErr error) (string, error) {
	path := ""
	err := prevErr

	errDelete := os.RemoveAll(dir)
	if errDelete != nil {
		path = dir
		if prevErr == nil {
			err = errDelete
		}
	}

	return path, err
}

func zipTarReader(r io.Reader) (*tar.Reader, error) {
	zipReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return tar.NewReader(zipReader), nil
}

func unTar(src *tar.Reader, dstPath string) error {
	for {
		header, err := src.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		dst := dstPath + "/" + header.Name
		mode := os.FileMode(header.Mode)
		switch header.Typeflag {
		case tar.TypeDir:
			err := os.MkdirAll(dst, mode)
			if err != nil {
				return err
			}
		case tar.TypeReg:
			err := makeFile(dst, mode, src)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unable to untar type : %c in file %s",
				header.Typeflag, header.Name)
		}
	}

	return nil
}

func makeFile(path string, mode os.FileMode, contents io.Reader) (err error) {
	writer, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		errClose := writer.Close()
		if err == nil {
			err = errClose
		}
	}()

	_, err = io.Copy(writer, contents)
	if err != nil {
		return err
	}

	if err = os.Chmod(path, mode); err != nil {
		return err
	}

	return nil
}
