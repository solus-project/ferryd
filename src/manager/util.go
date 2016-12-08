//
// Copyright © 2016 Ikey Doherty <ikey@solus-project.com>
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

package manager

import (
	"fmt"
	"io"
	"io/ioutil"
	"libeopkg"
	"os"
	"path/filepath"
	"strings"
)

var (
	// DiskSizeLabels provides a textual unit for the size of a Thing
	DiskSizeLabels = []string{
		"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB",
	}
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
	return nil
}

//
// FormPackageBasePath will return the appropriate path base for
// a package file to live in. In binman, we store packages with
// a file scheme similar to that of Debian. In all cases we use
// a lowercase source name, to prevent having "n", "N" directories.
//
//      $sourceFirstletter/$sourceName/$pkgfile
//      i.e:
//      g/glibc/glibc-2.24-35-1-x86_64.eopkg
//      g/glibc/glibc-32bit-2.24-35-1-x86_64.eopkg
//
// Special mutation is applied to names beginning with "lib", in that
// we shop off the first 4 letters.
//
//      libj/libjpeg-turbo/libjpeg-turbo-1.4.0-5-1-x86_64.eopkg
//
// This does also capture *non* library packages, such as LibreOffice,
// however it enforces a decent level of distribution of the package
// files among directories, which makes it much easier to navigate.
// It's also a pain to look at several thousand packages in a single
// directory..
func FormPackageBasePath(meta *libeopkg.Metadata) string {
	source := strings.ToLower(meta.Source.Name)
	if strings.HasPrefix(source, "lib") && len(source) > 3 {
		return filepath.Join(source[:4], source)
	}
	return filepath.Join(source[0:1], source)
}

// RemovePackageParents will try to remove the leading components of
// a package file, only if they are empty.
//
// Per FormPackageBasePath, every package is at a minimum depth of 2,
// and we clean those 2 dirs up if we can.
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

// IsValidName makes sure no path characters are used in a name,
// and that it contains no period characters (reserved for use in
// namespacing sub buckets)
func IsValidName(nom string) bool {
	for _, c := range nom {
		if c == '.' || c == '/' || c == '\\' || c == ';' {
			return false
		}
	}
	return true
}

// FormatDiskSizeUnit returns a size & unit appropriate for the input size
func FormatDiskSizeUnit(size int64) (float64, string) {
	fSize := float64(size)
	var i int
	var unit string

	for i, unit = range DiskSizeLabels {
		if fSize < 1024 || i == len(DiskSizeLabels)-1 {
			return fSize, unit
		}
		fSize /= 1024
	}

	return fSize, unit
}

// FormatDiskSize returns a textual size string for a given input size.
func FormatDiskSize(size int64) string {
	sz, unit := FormatDiskSizeUnit(size)
	return fmt.Sprintf("%.2f %s", sz, unit)
}

// GetFileSize is a quick utility function to get the size of any file.
func GetFileSize(path string) int64 {
	if st, err := os.Stat(path); st != nil && err == nil {
		return st.Size()
	}
	return -1
}
