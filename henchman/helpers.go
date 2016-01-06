package henchman

import (
	"archive/tar"
	_ "encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	_ "reflect"
	"strings"
)

// NOTE: This file is getting out of hand....

/**
 * These functions deal with merging of maps
 */
// source values will override dest values if override is true
// else dest values will not be overridden
func MergeMap(src map[interface{}]interface{}, dst map[interface{}]interface{}, override bool) {
	for variable, value := range src {
		if override == true {
			dst[variable] = value
		} else if _, present := dst[variable]; !present {
			dst[variable] = value
		}
	}
}

func MergeLogrusFields(src map[string]interface{}, dst map[string]interface{}, override bool) {
	for variable, value := range src {
		if override == true {
			dst[variable] = value
		} else if _, present := dst[variable]; !present {
			dst[variable] = value
		}
	}
}

/**
 * These functions deal with creating and removing temp dir and files
 */
// used to make tmp files in *_test.go
func createTempDir(folder string) string {
	name, _ := ioutil.TempDir("/tmp", folder)
	return name
}

func writeTempFile(buf []byte, fname string) string {
	fpath := path.Join("/tmp", fname)
	ioutil.WriteFile(fpath, buf, 0644)
	return fpath
}

func rmTempFile(fpath string) {
	os.Remove(fpath)
}

// Removes old dir and creates new one
func CreateDir(dir string) error {
	if err := os.Mkdir(dir, 0755); os.IsExist(err) {
		if err := os.RemoveAll(dir); err != nil {
			return HenchErr(err, nil, "While removing old dir")
		}
		if err := os.Mkdir(dir, 0755); err != nil {
			return HenchErr(err, nil, "Error creating dir")
		}
	}

	return nil
}

/**
 * These functions deal with tar
 */
// http://blog.ralch.com/tutorial/golang-working-with-tar-and-gzip/
// This function tars the source file/dir.  If a tar.Writer is detected
// it'll use that as the tar.Writer instead of generating it's own (this is useful
// when you want to have selective tarring, such as modules).
// Also if tarName is provided it'll use that
// returns the name of the target file (if created) and any errors
func tarit(source, tarName string, tb *tar.Writer) error {
	var tarball *tar.Writer
	target := tarName

	if tb != nil {
		tarball = tb
	} else {
		filename := filepath.Base(source)

		if target == "" {
			target = fmt.Sprintf("%s.tar", filename)
		}

		tarfile, err := os.Create(target)
		if err != nil {
			return HenchErr(err, nil, "While creating target")
		}
		defer tarfile.Close()

		tarball = tar.NewWriter(tarfile)
		defer tarball.Close()
	}

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	return filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return HenchErr(err, map[string]interface{}{
					"path": path,
				}, "While walking")
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return HenchErr(err, map[string]interface{}{
					"file":     path,
					"solution": "Golang specific tar package.  Submit an issue starting with TAR HEADER",
				}, "Adding info to tar header")
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			}

			if err := tarball.WriteHeader(header); err != nil {
				return HenchErr(err, map[string]interface{}{
					"file":     path,
					"solution": "Golang specific tar package.  Submit an issue starting with TARBALL",
				}, "Writing header to tar")
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return HenchErr(err, nil, "")
			}
			defer file.Close()

			if _, err := io.Copy(tarball, file); err != nil {
				return HenchErr(err, map[string]interface{}{
					"file":     path,
					"solution": "make sure file exists, correct permissions, or is not corrupted",
				}, "")
			}

			return nil
		})
}
