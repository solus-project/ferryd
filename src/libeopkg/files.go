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
	"os"
	"strconv"
)

// File is the idoimatic representation of the XML <File> node
//
// Note that directories are indicated by a missing hash. Unfortunately
// however eopkg doesn't record the actual _type_ of a file in an intelligent
// sense, thus we'll have to deal with symlinks separately.
//
// In an ideal world the package archive would be hash indexed with no file
// names or special permissions inside the archive, and we'd record all relevant
// metadata. This would allow a single copy, many hardlink approach to blit
// the files out, as well as allowing us to more accurately represent symbolic
// links instead of pretending they're real files.
//
// Long story short: Wait for eopkg's successor to worry about this stuff.
type File struct {
	Path      string
	Type      string
	Size      int64  `xml:"Size,omitempty"`
	UID       int    `xml:"Uid,omitempty"`
	GID       int    `xml:"Gid,omitempty"`
	Mode      string `xml:"Mode,omitempty"`
	Hash      string `xml:"Hash,omitempty"`
	Permanent string `xml:"Permanent,omitempty"`

	modePrivate os.FileMode // We populate this during files.xml read
}

// IsDir is a very trivial helper to determine if the file is meant to be a
// directory.
func (f *File) IsDir() bool {
	return f.Hash == ""
}

func (f *File) initFileMode() error {
	i, err := strconv.ParseUint(f.Mode, 8, 32)
	if err != nil {
		return err
	}
	f.modePrivate = os.FileMode(i)
	return nil
}

// FileMode will return an os.FileMode version of our string encoded "Mode" member
func (f *File) FileMode() os.FileMode {
	return f.modePrivate
}

// Files is the idiomatic representation of the XML <Files> node with one or
// more <File> children
type Files struct {
	File []*File
}
