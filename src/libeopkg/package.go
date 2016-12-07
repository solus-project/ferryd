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

package libeopkg

import (
	"archive/zip"
)

//
// A Package is used for accessing a `.eopkg` archive, the current format used
// within Solus for software packages.
//
// An .eopkg archive is actually a ZIP archive. Internally it has the following
// structure:
//
//      metadata.xml    -> Package information
//      files.xml       -> Record of the files and hash/uid/gid/etc
//      comar/          -> Postinstall scripts
//      install.tar.xz  -> Filesystem contents
//
// Due to this toplevel simplicity, we can use golang's native `archive/zip`
// library to achieve eopkg access, and parse the contents accordingly.
// This is much faster than having to call out to the host side tool, which
// is presently written in Python.
//
type Package struct {
	Path string // Path to this .eopkg file

	zipFile *zip.ReadCloser // .eopkg is a zip archvie
}

// Open will attempt to open the given .eopkg file.
// This must be a valid .eopkg file and this stage will assert that it is
// indeed a real archive.
func Open(path string) (*Package, error) {
	ret := &Package{
		Path: path,
	}
	zipFile, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	ret.zipFile = zipFile
	return ret, nil
}

// Close a previously opened .eopkg file
func (p *Package) Close() error {
	return p.zipFile.Close()
}
