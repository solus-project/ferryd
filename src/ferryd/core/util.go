//
// Copyright © 2016-2017 Solus Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package core

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"libeopkg"
	"os"
	"path/filepath"
)

// CopyFile will copy the file and permissions to the new target
func CopyFile(source, dest string) error {
	var src *os.File
	var dst *os.File
	var err error
	var st os.FileInfo

	// Stat the source first
	st, err = os.Stat(source)
	if err != nil {
		return nil
	}
	if src, err = os.Open(source); err != nil {
		return err
	}
	defer src.Close()
	if dst, err = os.OpenFile(dest, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, st.Mode()); err != nil {
		return err
	}
	// Copy the files
	if _, err = io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	dst.Close()
	// If it fails, meh.
	os.Chtimes(dest, st.ModTime(), st.ModTime())
	os.Chown(dest, os.Getuid(), os.Getgid())
	os.Chmod(dest, 00644)
	return nil
}

// LinkOrCopyFile is a helper which will initially try to hard link,
// however if we hit an error (because we tried a cross-filesystem hardlink)
// we'll try to copy instead.
func LinkOrCopyFile(source, dest string, forceCopy bool) error {
	if forceCopy {
		return CopyFile(source, dest)
	}
	if os.Link(source, dest) == nil {
		return nil
	}
	return CopyFile(source, dest)
}

// RemovePackageParents will try to remove the leading components of
// a package file, only if they are empty.
func RemovePackageParents(path string) error {
	sourceDir := filepath.Dir(path)      // i.e. libr/libreoffice
	letterDir := filepath.Dir(sourceDir) // i.e. libr/

	removalPaths := []string{
		sourceDir,
		letterDir,
	}

	for _, p := range removalPaths {
		contents, err := ioutil.ReadDir(p)
		if err != nil {
			return err
		}
		if len(contents) != 0 {
			continue
		}
		if err = os.Remove(p); err != nil {
			return err
		}
	}
	return nil
}

// AtomicRename will unlink the original path which will leave open file
// descriptors intact, and now position the new file into the old name, so
// that there is never a partial read on an index file.
func AtomicRename(origPath, newPath string) error {
	st, err := os.Stat(newPath)
	if err == nil && st.Mode().IsRegular() {
		if err = os.Remove(newPath); err != nil {
			return err
		}
	}
	return os.Rename(origPath, newPath)
}

// FileSha1sum is a quick wrapper to grab the sha1sum for the given file
func FileSha1sum(path string) (string, error) {
	mfile, err := MapFile(path)
	if err != nil {
		return "", err
	}
	defer mfile.Close()
	h := sha1.New()
	// Pump from memory into hash for zero-copy sha1sum
	h.Write(mfile.Data)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// FileSha256sum is a quick wrapper to grab the sha256sum for the given file
func FileSha256sum(path string) (string, error) {
	mfile, err := MapFile(path)
	if err != nil {
		return "", err
	}
	defer mfile.Close()
	h := sha256.New()
	// Pump from memory into hash for zero-copy sha1sum
	h.Write(mfile.Data)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// WriteSha1sum will take the sha1sum of the input path and then dump it to
// the given output path
func WriteSha1sum(inpPath, outPath string) error {
	hash, err := FileSha1sum(inpPath)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(outPath, []byte(hash), 00644)
}

// PathExists is a trivial helper to figure out if a path exists or not
func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

// ProduceDelta will attempt to batch the delta production between the
// two listed file paths and then copy it into the final targetPath
func ProduceDelta(tmpDir, oldPackage, newPackage, targetPath string) error {
	del, err := libeopkg.NewDeltaProducer(tmpDir, oldPackage, newPackage)
	if err != nil {
		return err
	}
	defer del.Close()
	path, err := del.Commit()
	if err != nil {
		return err
	}

	// Always nuke the tmpfile
	defer os.Remove(path)

	return LinkOrCopyFile(path, targetPath, false)
}
