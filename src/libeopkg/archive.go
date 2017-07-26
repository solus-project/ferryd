//
// Copyright © 2017 Ikey Doherty <ikey@solus-project.com>
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

package libeopkg

import (
	"archive/tar"
	"io"
)

// ArchiveReader is used to allow reading directly from an install.tar.xz
// file with the purpose of pulling out records, etc.
type ArchiveReader struct {
	pkg     *Package
	tarfile *tar.Reader
	fi      io.ReadCloser
}

// NewArchiveReader will return a new install.tar.xz reader for the given package
func NewArchiveReader(pkg *Package) (*ArchiveReader, error) {
	r := &ArchiveReader{
		pkg: pkg,
	}

	// Ensure files.xml/metadata.xml is actually read
	if err := pkg.ReadAll(); err != nil {
		return nil, err
	}

	// Make sure install.tar.xz is really present!
	contents := pkg.FindFile("install.tar.xz")
	if contents == nil {
		return nil, ErrEopkgCorrupted
	}

	fi, err := contents.Open()
	if err != nil {
		return nil, err
	}
	r.fi = fi

	// TODO: Stick xzed read here, because its not going to parse right now!
	r.tarfile = tar.NewReader(fi)
	return r, nil
}

// Close the ArchiveReader and any underlying resources
func (a *ArchiveReader) Close() {
	if a.tarfile == nil {
		return
	}
	a.fi.Close()
	a.tarfile = nil
}
