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
	"bytes"
	"encoding/gob"
	"github.com/boltdb/bolt"
	"libeopkg"
	"os"
	"path/filepath"
	"sort"
)

// addSourcePackage will add the package ID to the source mapping contained inside
// the soruce bucket.
// This function must always be called from a transaction
func (m *Manager) addSourcePackage(repo *Repository, tx *bolt.Tx, pkgID string, meta *libeopkg.Metadata) error {
	bucket := tx.Bucket(BucketNameRepos).Bucket(repo.BucketPathSources())
	var sourceMapping []string

	buf := &bytes.Buffer{}

	source := []byte(meta.Source.Name)

	// Deserialise the current slice of IDs
	bits := bucket.Get(source)
	if len(bits) != 0 {
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(&sourceMapping); err != nil {
			return err
		}
	}

	// Ensure the package isn't already here.
	for _, id := range sourceMapping {
		if id == pkgID {
			return ErrResourceExists
		}
	}
	sourceMapping = append(sourceMapping, pkgID)
	// Preserve an order to prevent popping holes in the DB
	sort.Strings(sourceMapping)

	buf.Reset()
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&sourceMapping); err != nil {
		return err
	}
	return bucket.Put(source, buf.Bytes())
}

// addPackageToRepo will take care of internalising this package into the
// given repository, and exposing the file on the repo filesystem.
func (m *Manager) addPackageToRepo(repo *Repository, pkg *libeopkg.Package, poolPath string) error {
	// From -> to vars
	repoDir := filepath.Join(m.rootDir, repo.GetDirectory())
	tgtDir := filepath.Join(repoDir, FormPackageBasePath(pkg.Meta))
	tgtPath := filepath.Join(tgtDir, filepath.Base(pkg.Path))

	// Create leading directory structure
	if err := os.MkdirAll(tgtDir, 00755); err != nil {
		return err
	}

	// Now store it into the db
	err := m.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketNameRepos).Bucket(repo.BucketPathPackages())
		// Perhaps store this in the pkg object.. ?
		pkgID := []byte(filepath.Base(pkg.Path))
		// Ensure the package isn't already stored!
		if len(bucket.Get(pkgID)) != 0 {
			return ErrResourceExists
		}
		// Stick the source mapping in
		if err := m.addSourcePackage(repo, tx, string(pkgID), pkg.Meta); err != nil {
			return err
		}
		return bucket.Put(pkgID, []byte(pkg.Meta.Source.Name))
	})

	// if DB failed, nuke package parents
	if err != nil {
		defer RemovePackageParents(tgtPath)
		return err
	}

	// Attempt to link the path into place, nuke parents if this fails
	if err := os.Link(poolPath, tgtPath); err != nil {
		defer RemovePackageParents(tgtPath)
		return err
	}
	return nil
}

//
// AddPackage will try to add a single package to the given repo.
//
// Initially the new package will be referenced, which will then move that
// package into pool/ area if currently unknown. If for any reason there
// is an area pushing the package into the repository, it will automatically
// be deref'd.
// This ensures there are no "stragglers" left on the filesystem, as botched
// uploads will immediately evaporate into thin air.
func (m *Manager) AddPackage(reponame string, pkgPath string) error {
	repo, err := m.GetRepo(reponame)
	if err != nil {
		return err
	}
	baseName := filepath.Base(pkgPath)
	var poolPath string

	pkg, err := libeopkg.Open(pkgPath)
	if err != nil {
		return err
	}
	defer pkg.Close()
	// Load only the metadata at this point
	if err := pkg.ReadMetadata(); err != nil {
		return err
	}
	// First things first, try to ref the package
	if poolPath, err = m.pool.RefPackage(pkg); err != nil {
		return err
	}
	if err = m.addPackageToRepo(repo, pkg, poolPath); err != nil {
		defer m.pool.UnrefPackage(baseName)
		return err
	}
	return nil
}

// AddPackages will add all of the given packages to the specified resource
func (m *Manager) AddPackages(repoName string, pkgs []string) error {
	// TODO: Check the repo exists!

	// Iterate and open all of the packages
	for _, pkgPath := range pkgs {
		if err := m.AddPackage(repoName, pkgPath); err != nil {
			return err
		}
	}
	return nil
}
