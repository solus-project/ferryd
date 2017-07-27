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
	"errors"
)

// DeltaProducer is responsible for taking two eopkg packages and spitting out
// a delta package for them, containing only the new files.
type DeltaProducer struct {
	old *ArchiveReader
	new *ArchiveReader
}

var (
	// ErrMismatchedDelta is returned when the input packages should never be delta'd,
	// i.e. they're unrelated
	ErrMismatchedDelta = errors.New("Delta is not possible between the input packages")
)

// NewDeltaProducer will return a new delta producer for the given input packages
// It is very important that the old and new packages are in the correct order!
func NewDeltaProducer(pkgOld string, pkgNew string) (*DeltaProducer, error) {
	var err error
	ret := &DeltaProducer{}
	defer func() {
		if err != nil {
			ret.Close()
		}
	}()
	ret.old, err = NewArchiveReaderFromFilename(pkgOld)
	if err != nil {
		return nil, err
	}

	ret.new, err = NewArchiveReaderFromFilename(pkgNew)
	if err != nil {
		return nil, err
	}

	if !IsDeltaPossible(&ret.old.pkg.Meta.Package, &ret.new.pkg.Meta.Package) {
		return nil, ErrMismatchedDelta
	}

	return ret, nil
}

// Close the DeltaProducer
func (d *DeltaProducer) Close() {
	if d.old != nil {
		d.old.Close()
		d.old = nil
	}
	if d.new != nil {
		d.new.Close()
		d.new = nil
	}
}

// filesToMap is a helper that will let us uniquely index hash to file-set
func (d *DeltaProducer) filesToMap(r *ArchiveReader) (ret map[string][]*File) {
	ret = make(map[string][]*File)
	for _, f := range r.pkg.Files.File {
		if _, ok := ret[f.Hash]; !ok {
			ret[f.Hash] = ([]*File{f})
		} else {
			ret[f.Hash] = append(ret[f.Hash], f)
		}
	}
	return ret
}

// Commit will attempt to produce a delta between the 2 eopkg files
func (d *DeltaProducer) Commit() error {
	hashOldFiles := d.filesToMap(d.old)
	hashNewFiles := d.filesToMap(d.new)

	diffMap := make(map[string]int)

	// Note this is very simple and works just like the existing eopkg functionality
	// which is purely hash-diff based. eopkg will look for relocations on applying
	// the update so that files get "reused"
	for h, s := range hashNewFiles {
		if _, ok := hashOldFiles[h]; ok {
			continue
		}
		for _, p := range s {
			diffMap[p.Path] = 1
		}
	}

	// TODO: Create a new temporary install.tar.xz, and copy all data from the
	// original install.tar.xz into it. Then wrap an eopkg around it
	return ErrMismatchedDelta
}
