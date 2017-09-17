//
// Copyright © 2017 Solus Project
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

package core

import (
	"fmt"
	"github.com/boltdb/bolt"
	"libeopkg"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const (
	// RepoPathComponent is the base for all repository directories
	RepoPathComponent = "repo"

	// AssetPathComponent is where we'll find extra files like distribution.xml
	AssetPathComponent = "assets"

	// DeltaPathComponent is a temporary tree for creating delta packages
	DeltaPathComponent = "deltaBuilds"

	// DeltaStagePathComponent is where we put temporary deltas until merged
	DeltaStagePathComponent = "deltaStaging"

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
	repoBase       string
	assetBase      string
	deltaBase      string
	deltaStageBase string
	transcoder     *GobTranscoder

	repos map[string]*Repository // Cache all repositories.
}

// A Repository is a simplistic representation of a exported repository
// within ferryd
type Repository struct {
	ID             string                 // Name of this repository (unique)
	path           string                 // Where this is on disk
	assetPath      string                 // Where our assets are stored on disk
	deltaPath      string                 // Where we'll produce deltas
	deltaStagePath string                 // Where we'll stage final deltas
	dist           *libeopkg.Distribution // Distribution

	mut *sync.RWMutex // Allow locking a repository for inserts and indexes
}

// RepoEntry is the basic repository storage unit, and details what packages
// are exported in the index.
type RepoEntry struct {
	SchemaVersion string   // Version used when this repo entry was created
	Name          string   // Base package name
	Available     []string // The available packages for this package name (eopkg IDs)
	Published     string   // The "tip" version of this package (eopkg ID)
	Deltas        []string // Delta packages known for this package.
}

// Init will create our initial working paths and DB bucket
func (r *RepositoryManager) Init(ctx *Context, tx *bolt.Tx) error {
	r.repoBase = filepath.Join(ctx.BaseDir, RepoPathComponent)
	r.assetBase = filepath.Join(ctx.BaseDir, AssetPathComponent)
	r.deltaBase = filepath.Join(ctx.BaseDir, DeltaPathComponent)
	r.deltaStageBase = filepath.Join(ctx.BaseDir, DeltaStagePathComponent)
	r.transcoder = NewGobTranscoder()
	r.repos = make(map[string]*Repository)

	paths := []string{
		r.repoBase,
		r.assetBase,
		r.deltaBase,
	}
	// Ensure we have all paths
	for _, p := range paths {
		if err := os.MkdirAll(p, 00755); err != nil {
			return err
		}
	}
	_, err := tx.CreateBucketIfNotExists([]byte(DatabaseBucketRepo))
	return err
}

// Close doesn't currently do anything
func (r *RepositoryManager) Close() {}

// bakeRepo hands the internal duped code between GetRepo/CreateRepo to ensure
// they're always fully formed.
//
// This ensures the first time we GetRepo on an existing repo, we ensure that
// we actually have all support paths too.
func (r *RepositoryManager) bakeRepo(id string) (*Repository, error) {
	repository := &Repository{
		ID:             id,
		path:           filepath.Join(r.repoBase, id),
		assetPath:      filepath.Join(r.assetBase, id),
		deltaPath:      filepath.Join(r.deltaBase, id),
		deltaStagePath: filepath.Join(r.deltaStageBase, id),
		mut:            &sync.RWMutex{},
	}

	paths := []string{
		repository.path,
		repository.assetPath,
		repository.deltaPath,
		repository.deltaStagePath,
	}

	// Create all required paths
	for _, p := range paths {
		if PathExists(p) {
			continue
		}
		if err := os.MkdirAll(p, 00755); err != nil {
			return nil, err
		}
	}

	return repository, nil
}

// GetRepo will attempt to get the named repo if it exists, otherwise
// return an error. This is a transactional helper to make the API simpler
func (r *RepositoryManager) GetRepo(tx *bolt.Tx, id string) (*Repository, error) {
	// Cache each repository.
	if repo, ok := r.repos[id]; ok {
		return repo, nil
	}

	rootBucket := tx.Bucket([]byte(DatabaseBucketRepo))
	repo := rootBucket.Bucket([]byte(id))
	if repo == nil {
		return nil, fmt.Errorf("The specified repository '%s' does not exist", id)
	}

	repository, err := r.bakeRepo(id)
	if err != nil {
		return nil, err
	}

	// Cache this guy for later
	r.repos[id] = repository

	return repository, nil
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

	repository, err := r.bakeRepo(id)
	if err != nil {
		return nil, err
	}

	// Cache this guy for later
	r.repos[id] = repository

	return repository, nil
}

// GetEntry will return the package entry for the given ID
func (r *Repository) GetEntry(tx *bolt.Tx, id string) (*RepoEntry, error) {
	r.mut.RLock()
	defer r.mut.RUnlock()

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
	r.mut.Lock()
	defer r.mut.Unlock()

	rootBucket := tx.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))
	code := NewGobEncoderLight()
	enc, err := code.EncodeType(entry)
	if err != nil {
		return err
	}

	return rootBucket.Put([]byte(entry.Name), enc)
}

// AddDelta will first open and read the .delta.eopkg, before passing it back off to AddLocalDelta
func (r *Repository) AddDelta(tx *bolt.Tx, pool *Pool, filename string, mapping *DeltaInformation) error {
	pkg, err := libeopkg.Open(filename)
	if err != nil {
		return err
	}

	defer pkg.Close()
	if err = pkg.ReadMetadata(); err != nil {
		return err
	}

	return r.AddLocalDelta(tx, pool, pkg, mapping)
}

// AddLocalDelta will attempt to add the delta to this repository, if possible
// All ref'd deltas are retained, but not necessarily emitted unless they're
// valid for the from-to relationship.
func (r *Repository) AddLocalDelta(tx *bolt.Tx, pool *Pool, pkg *libeopkg.Package, mapping *DeltaInformation) error {
	// Find our local package entry for the delta package first
	entry, err := r.GetEntry(tx, pkg.Meta.Package.Name)
	if err != nil {
		return err
	}

	pkgDir := filepath.Join(r.path, pkg.Meta.Package.GetPathComponent())
	pkgTarget := filepath.Join(pkgDir, pkg.ID)

	// Check we don't know about this delta already
	for _, id := range entry.Deltas {
		if id == pkg.ID {
			fmt.Printf("Skipping already included delta %s\n", id)
			return nil
		}
	}

	// Insert this deltas ID to this package map
	entry.Deltas = append(entry.Deltas, pkg.ID)
	sort.Strings(entry.Deltas)

	// Construct root dirs
	if err := os.MkdirAll(pkgDir, 00755); err != nil {
		return err
	}

	// Grab the pool reference for this package
	if _, err = pool.AddDelta(tx, pkg, mapping, false); err != nil {
		return err
	}

	// Ensure the eopkg file is linked inside our own tree
	source := pool.GetPackagePoolPath(pkg)
	if err = LinkOrCopyFile(source, pkgTarget, false); err != nil {
		return err
	}

	return r.putEntry(tx, entry)
}

// AddLocalPackage will do the real work of adding an open & loaded eopkg to the repository
func (r *Repository) AddLocalPackage(tx *bolt.Tx, pool *Pool, pkg *libeopkg.Package) error {
	repoEntry := &RepoEntry{
		SchemaVersion: RepoSchemaVersion,
		Name:          pkg.Meta.Package.Name,
		Published:     pkg.ID,
	}

	pkgDir := filepath.Join(r.path, pkg.Meta.Package.GetPathComponent())
	pkgTarget := filepath.Join(pkgDir, pkg.ID)

	// Already have a package, so let's copy the existing bits over
	entry, err := r.GetEntry(tx, pkg.Meta.Package.Name)
	if entry != nil {
		repoEntry.Available = entry.Available
		repoEntry.Published = entry.Published

		pkgAvail, err := pool.GetEntry(tx, repoEntry.Published)
		if err == nil {
			if pkg.Meta.Package.GetRelease() > pkgAvail.Meta.GetRelease() {
				repoEntry.Published = pkg.ID
			} else if pkg.Meta.Package.GetRelease() == pkgAvail.Meta.GetRelease() && pkgAvail.Name != pkg.ID {
				fmt.Printf(" **** DUPLICATE RELEASE NUMBER DETECTED. FAK: %s %s **** \n", pkg.ID, pkgAvail.Name)
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
	if _, err = pool.AddPackage(tx, pkg, false); err != nil {
		return err
	}

	// Ensure the eopkg file is linked inside our own tree
	source := pool.GetPackagePoolPath(pkg)
	if err = LinkOrCopyFile(source, pkgTarget, false); err != nil {
		return err
	}

	return r.putEntry(tx, repoEntry)
}

// AddPackage will attempt to load the local package and then add it to the
// repository via AddLocalPackage
func (r *Repository) AddPackage(tx *bolt.Tx, pool *Pool, filename string) error {
	pkg, err := libeopkg.Open(filename)
	if err != nil {
		return err
	}

	defer pkg.Close()
	if err = pkg.ReadMetadata(); err != nil {
		return err
	}

	return r.AddLocalPackage(tx, pool, pkg)
}

// GetPackageNames will traverse the buckets and find all package names as stored
// within the DB. This doesn't account for obsolete names, which should in fact
// be removed from the repo entirely.
func (r *Repository) GetPackageNames(tx *bolt.Tx) ([]string, error) {
	r.mut.RLock()
	defer r.mut.RUnlock()

	var pkgIds []string
	rootBucket := tx.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))

	c := rootBucket.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		code := NewGobDecoderLight()
		entry := RepoEntry{}
		if err := code.DecodeType(v, &entry); err != nil {
			return nil, err
		}

		pkgIds = append(pkgIds, entry.Name)
	}
	return pkgIds, nil
}

// GetPackages will return all package objects for a given name
func (r *Repository) GetPackages(tx *bolt.Tx, pool *Pool, pkgName string) ([]*libeopkg.MetaPackage, error) {
	r.mut.RLock()
	defer r.mut.RUnlock()

	var pkgs []*libeopkg.MetaPackage

	entry, err := r.GetEntry(tx, pkgName)
	if err != nil || entry == nil {
		return nil, err
	}

	for _, id := range entry.Available {
		p, err := pool.GetEntry(tx, id)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p.Meta)
	}

	return pkgs, nil
}

// CreateDelta is responsible for trying to create a new delta package between
// oldPkg and newPkg, with newPkg being the delta *to*.
//
// This function may fail to produce a delta because they're incompatible packages,
// or because a delta between the two packages would be pointless (i.e. they're
// either identical or 100% the same.)
//
// Lastly, this function will move the delta out of the build area into the
// staging area if it successfully produces a delta. This does not mark a delta
// attempt as "pointless", nor does it actually *include* the delta package
// within the repository.
func (r *Repository) CreateDelta(tx *bolt.Tx, oldPkg, newPkg *libeopkg.MetaPackage) (string, error) {
	if !libeopkg.IsDeltaPossible(oldPkg, newPkg) {
		return "", libeopkg.ErrMismatchedDelta
	}
	fileName := libeopkg.ComputeDeltaName(oldPkg, newPkg)
	fullPath := filepath.Join(r.deltaStagePath, fileName)

	// This guy exists, no point in trying to rebuild it
	if PathExists(fullPath) {
		return fullPath, nil
	}

	oldPath := filepath.Join(r.path, oldPkg.PackageURI)
	newPath := filepath.Join(r.path, newPkg.PackageURI)

	fmt.Printf(" * Forming delta %s\n", fullPath)

	if err := ProduceDelta(r.deltaPath, oldPath, newPath, fullPath); err != nil {
		return "", err
	}

	return fullPath, nil
}
