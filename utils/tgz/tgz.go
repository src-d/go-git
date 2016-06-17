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
// On success, the path of newly created directory and a nil error is
// returned. Otherwise an error is returned along with the path of the
// newly created directory with whatever information was extracted
// before the error or a empty string if no directory was created.
func Extract(tgz string) (d string, err error) {
	f, err := os.Open(tgz)
	if err != nil {
		return "", err
	}

	defer func() {
		errClose := f.Close()
		if err == nil {
			err = errClose
		}
	}()

	d, err = ioutil.TempDir(useDefaultTempDir, tmpPrefix)
	if err != nil {
		return "", nil
	}

	tar, err := zipTarReader(f)
	if err != nil {
		return d, err
	}

	if err = unTar(tar, d); err != nil {
		return d, err
	}

	return d, nil
}

func zipTarReader(r io.Reader) (*tar.Reader, error) {
	zip, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return tar.NewReader(zip), nil
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
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		errClose := w.Close()
		if err == nil {
			err = errClose
		}
	}()

	_, err = io.Copy(w, contents)
	if err != nil {
		return err
	}

	if err = os.Chmod(path, mode); err != nil {
		return err
	}

	return nil
}
