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

package libferry

import (
	"fmt"
	"github.com/boltdb/bolt"
	"libeopkg"
	"os"
	"path/filepath"
	"sort"
)

const (
	// RepoPathComponent is the base for all repository directories
	RepoPathComponent = "repo"

	// DatabaseBucketRepo is the name for the main repo toplevel bucket
	DatabaseBucketRepo = "repo"

	// DatabaseBucketPackage is the path to the subbucket within a repo bucket
	DatabaseBucketPackage = "package"

	// RepoSchemaVersion is the current schema version for a RepoEntry
	RepoSchemaVersion = "1.0"
)

// The RepositoryManager maintains all repos within ferryd which are in
// turn linked to the main pool
type RepositoryManager struct {
	repoBase   string
	transcoder *GobTranscoder
}

// A Repository is a simplistic representation of a exported repository
// within ferryd
type Repository struct {
	ID   string // Name of this repository (unique)
	path string // Where this is on disk
}

// RepoEntry is the basic repository storage unit, and details what packages
// are exported in the index.
type RepoEntry struct {
	SchemaVersion string   // Version used when this repo entry was created
	Name          string   // Base package name
	Available     []string // The available packages for this package name (eopkg IDs)
	Published     string   // The "tip" version of this package (eopkg ID)
}

// Init will create our initial working paths and DB bucket
func (r *RepositoryManager) Init(ctx *Context, tx *bolt.Tx) error {
	r.repoBase = filepath.Join(ctx.BaseDir, RepoPathComponent)
	r.transcoder = NewGobTranscoder()
	if err := os.MkdirAll(r.repoBase, 00755); err != nil {
		return err
	}
	_, err := tx.CreateBucketIfNotExists([]byte(DatabaseBucketRepo))
	return err
}

// Close doesn't currently do anything
func (r *RepositoryManager) Close() {}

// GetRepo will attempt to get the named repo if it exists, otherwise
// return an error. This is a transactional helper to make the API simpler
func (r *RepositoryManager) GetRepo(tx *bolt.Tx, id string) (*Repository, error) {
	rootBucket := tx.Bucket([]byte(DatabaseBucketRepo))
	repo := rootBucket.Bucket([]byte(id))
	if repo == nil {
		return nil, fmt.Errorf("The specified repository '%s' does not exist", id)
	}
	return &Repository{
		ID:   id,
		path: filepath.Join(r.repoBase, id),
	}, nil
}

// CreateRepo will create a new repository (bucket) within the top level
// repo bucket.
func (r *RepositoryManager) CreateRepo(tx *bolt.Tx, id string) (*Repository, error) {
	if _, err := r.GetRepo(tx, id); err == nil {
		return nil, fmt.Errorf("The specified repository '%s' already exists", id)
	}

	// Create the main sub-bucket for this repo
	rootBucket := tx.Bucket([]byte(DatabaseBucketRepo))
	bucket, err := rootBucket.CreateBucket([]byte(id))
	if err != nil {
		return nil, err
	}

	// Storage for package entries
	_, err = bucket.CreateBucket([]byte(DatabaseBucketPackage))
	if err != nil {
		return nil, err
	}

	repoDir := filepath.Join(r.repoBase, id)
	if err := os.MkdirAll(repoDir, 00755); err != nil {
		return nil, err
	}
	return &Repository{
		ID:   id,
		path: repoDir,
	}, nil
}

// GetEntry will return the package entry for the given ID
func (r *Repository) GetEntry(tx *bolt.Tx, id string) (*RepoEntry, error) {
	rootBucket := tx.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))
	v := rootBucket.Get([]byte(id))
	if v == nil {
		return nil, nil
	}
	entry := &RepoEntry{}
	code := NewGobDecoderLight()
	if err := code.DecodeType(v, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// Private method to re-put the entry into the DB
func (r *Repository) putEntry(tx *bolt.Tx, entry *RepoEntry) error {
	rootBucket := tx.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))
	code := NewGobEncoderLight()
	enc, err := code.EncodeType(entry)
	if err != nil {
		return err
	}

	return rootBucket.Put([]byte(entry.Name), enc)
}

// AddPackage will attempt to add the package to this repository
func (r *Repository) AddPackage(tx *bolt.Tx, pool *Pool, filename string) error {
	pkg, err := libeopkg.Open(filename)
	if err != nil {
		return err
	}

	defer pkg.Close()
	if err = pkg.ReadMetadata(); err != nil {
		return err
	}

	pkgDir := filepath.Join(r.path, pkg.Meta.Package.GetPathComponent())
	pkgTarget := filepath.Join(pkgDir, pkg.ID)

	fmt.Printf("Processing %s-%s-%d\n", pkg.Meta.Package.Name, pkg.Meta.Package.GetVersion(), pkg.Meta.Package.GetRelease())

	repoEntry := &RepoEntry{
		SchemaVersion: RepoSchemaVersion,
		Name:          pkg.Meta.Package.Name,
		Published:     pkg.ID,
	}

	// Already have a package, so let's copy the existing bits over
	entry, err := r.GetEntry(tx, pkg.Meta.Package.Name)
	if entry != nil {
		repoEntry.Available = entry.Available
		repoEntry.Published = entry.Published

		pkgAvail, err := pool.GetEntry(tx, repoEntry.Published)
		if err == nil {
			if pkg.Meta.Package.GetRelease() > pkgAvail.Meta.GetRelease() {
				repoEntry.Published = pkg.ID
			}
		} else {
			repoEntry.Published = pkg.ID
		}
	} else if err != nil {
		fmt.Printf("Error was %v\n", err)
	}

	// Check if we've already indexed it, non-fatal
	for _, id := range repoEntry.Available {
		if id == pkg.ID {
			fmt.Printf("Skipping already included %s\n", id)
			return nil
		}
	}

	// Construct root dirs
	if err := os.MkdirAll(pkgDir, 00755); err != nil {
		return err
	}

	// Keep the available list clean + sorted
	repoEntry.Available = append(repoEntry.Available, pkg.ID)
	sort.Strings(repoEntry.Available)

	// Grab the pool reference for this package (Always copy)
	if _, err = pool.AddPackage(tx, pkg, true); err != nil {
		return err
	}

	// Ensure the eopkg file is linked inside our own tree
	source := pool.GetPackagePoolPath(pkg)
	if err = LinkOrCopyFile(source, pkgTarget, false); err != nil {
		return err
	}

	return r.putEntry(tx, repoEntry)
}
