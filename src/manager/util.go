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
	"io"
	"libeopkg"
	"os"
	"path/filepath"
	"strings"
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
