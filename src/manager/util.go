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
	"os"
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
