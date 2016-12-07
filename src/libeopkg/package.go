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

// Package represents a binary .eopkg file
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
