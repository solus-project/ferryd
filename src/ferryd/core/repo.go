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
	"libdb"
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

	repoLock *sync.Mutex

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

	indexMut *sync.Mutex // Indexing requires a special, separate lock
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
func (r *RepositoryManager) Init(ctx *Context, db libdb.Database) error {
	r.repoBase = filepath.Join(ctx.BaseDir, RepoPathComponent)
	r.assetBase = filepath.Join(ctx.BaseDir, AssetPathComponent)
	r.deltaBase = filepath.Join(ctx.BaseDir, DeltaPathComponent)
	r.deltaStageBase = filepath.Join(ctx.BaseDir, DeltaStagePathComponent)
	r.repoLock = &sync.Mutex{}
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
	return nil
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
		indexMut:       &sync.Mutex{},
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

// GetRepos will return a copy of the repositores in our database
func (r *RepositoryManager) GetRepos(db libdb.Database) ([]*Repository, error) {
	var ret []*Repository
	err := db.Bucket([]byte(DatabaseBucketRepo)).View(func(db libdb.ReadOnlyView) error {
		return db.ForEach(func(key, value []byte) error {
			var repo Repository
			if err := db.Decode(value, &repo); err != nil {
				return err
			}
			ret = append(ret, &repo)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// GetRepo will attempt to get the named repo if it exists, otherwise
// return an error. This is a transactional helper to make the API simpler
func (r *RepositoryManager) GetRepo(db libdb.Database, id string) (*Repository, error) {
	// Cache each repository.
	if repo, ok := r.repos[id]; ok {
		return repo, nil
	}

	var rTmp Repository
	rootBucket := db.Bucket([]byte(DatabaseBucketRepo))
	if err := rootBucket.GetObject([]byte(id), &rTmp); err != nil {
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
func (r *RepositoryManager) CreateRepo(db libdb.Database, id string) (*Repository, error) {
	r.repoLock.Lock()
	defer r.repoLock.Unlock()

	if _, err := r.GetRepo(db, id); err == nil {
		return nil, fmt.Errorf("The specified repository '%s' already exists", id)
	}

	// Create the main sub-bucket for this repo
	rootBucket := db.Bucket([]byte(DatabaseBucketRepo))
	repo := Repository{
		ID: id,
	}

	if err := rootBucket.PutObject([]byte(id), &repo); err != nil {
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

// DeleteRepo is not yet implemented
func (r *RepositoryManager) DeleteRepo(db libdb.Database, pool *Pool, id string) error {
	r.repoLock.Lock()
	defer r.repoLock.Unlock()

	repo, err := r.GetRepo(db, id)
	if err != nil {
		return fmt.Errorf("The specified repository '%s' does not exist", id)
	}

	// TODO: lock this repository from accepting inclusions ..
	delete(r.repos, id)

	// Let's iterate over every one of our packages here and start up an unref
	// cycle
	err = db.Update(func(db libdb.Database) error {
		repoBucket := db.Bucket([]byte(DatabaseBucketRepo))
		rootBucket := repoBucket.Bucket([]byte(repo.ID)).Bucket([]byte(DatabaseBucketPackage))

		err := rootBucket.ForEach(func(k, v []byte) error {
			// Grab each repo entry
			entry := RepoEntry{}
			if err := rootBucket.Decode(v, &entry); err != nil {
				return err
			}

			// First up, find all the packages to unref
			for _, id := range entry.Available {
				if err := repo.removePackageInternal(db, pool, id); err != nil {
					return err
				}
			}

			// Next up, find all the deltas to unref
			for _, id := range entry.Deltas {
				if err := repo.removeDeltaInternal(db, pool, id); err != nil {
					return err
				}
			}

			// Remove all IDs for this package, now we must remove this entry
			// Note this isn't applied within the foreach.
			return rootBucket.DeleteObject([]byte(entry.Name))
		})
		if err != nil {
			return err
		}
		// Now remove the repository object itself
		return repoBucket.DeleteObject([]byte(repo.ID))
	})

	if err != nil {
		return err
	}

	deletionPaths := []string{
		repo.path,
		repo.assetPath,
		repo.deltaPath,
		repo.deltaStagePath,
	}

	// Clean up the repo paths.
	// TODO: Make non fatal, just attempt what we can.

	for _, p := range deletionPaths {
		if !PathExists(p) {
			continue
		}
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}

	return nil
}

// GetEntry will return the package entry for the given ID
func (r *Repository) GetEntry(db libdb.Database, id string) (*RepoEntry, error) {
	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))
	entry := &RepoEntry{}
	if err := rootBucket.GetObject([]byte(id), entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// Private method to re-put the entry into the DB
//
// TODO: Consider write protection
func (r *Repository) putEntry(db libdb.Database, entry *RepoEntry) error {
	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))
	return rootBucket.PutObject([]byte(entry.Name), entry)
}

// AddDelta will first open and read the .delta.eopkg, before passing it back off to AddLocalDelta
func (r *Repository) AddDelta(db libdb.Database, pool *Pool, filename string, mapping *DeltaInformation) error {
	pkg, err := libeopkg.Open(filename)
	if err != nil {
		return err
	}

	defer pkg.Close()
	if err = pkg.ReadMetadata(); err != nil {
		return err
	}

	return r.AddLocalDelta(db, pool, pkg, mapping)
}

// AddLocalDelta will attempt to add the delta to this repository, if possible
// All ref'd deltas are retained, but not necessarily emitted unless they're
// valid for the from-to relationship.
func (r *Repository) AddLocalDelta(db libdb.Database, pool *Pool, pkg *libeopkg.Package, mapping *DeltaInformation) error {
	// Find our local package entry for the delta package first
	entry, err := r.GetEntry(db, pkg.Meta.Package.Name)
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
	if _, err = pool.AddDelta(db, pkg, mapping, false); err != nil {
		return err
	}

	// Ensure the eopkg file is linked inside our own tree
	source := pool.GetPackagePoolPath(pkg)
	if err = LinkOrCopyFile(source, pkgTarget, false); err != nil {
		return err
	}

	return r.putEntry(db, entry)
}

// Internal helper to remove packages
func (r *Repository) removePackageInternal(db libdb.Database, pool *Pool, id string) error {
	poolEntry, err := pool.GetEntry(db, id)
	if err != nil {
		return nil
	}

	pkgDir := filepath.Join(r.path, poolEntry.Meta.GetPathComponent())
	pkgTarget := filepath.Join(pkgDir, poolEntry.Name)

	// TODO: Consider making non-fatal..
	if err = os.Remove(pkgTarget); err != nil {
		return err
	}

	// TODO: This likely shouldn't be fatal either
	if err = RemovePackageParents(pkgTarget); err != nil {
		return err
	}

	// Tell the pool we no longer need this guy
	return pool.UnrefEntry(db, id)
}

// removeDeltaInternal has the same job as removePackageInternal, but in future should
// be extended to remove the skip records
func (r *Repository) removeDeltaInternal(db libdb.Database, pool *Pool, id string) error {
	return r.removePackageInternal(db, pool, id)
}

// AddLocalPackage will do the real work of adding an open & loaded eopkg to the repository
func (r *Repository) AddLocalPackage(db libdb.Database, pool *Pool, pkg *libeopkg.Package) error {
	repoEntry := &RepoEntry{
		SchemaVersion: RepoSchemaVersion,
		Name:          pkg.Meta.Package.Name,
		Published:     pkg.ID,
	}

	pkgDir := filepath.Join(r.path, pkg.Meta.Package.GetPathComponent())
	pkgTarget := filepath.Join(pkgDir, pkg.ID)

	// Already have a package, so let's copy the existing bits over
	entry, err := r.GetEntry(db, pkg.Meta.Package.Name)
	if entry != nil {
		repoEntry.Available = entry.Available
		repoEntry.Published = entry.Published

		pkgAvail, err := pool.GetEntry(db, repoEntry.Published)
		if err == nil {
			if pkg.Meta.Package.GetRelease() > pkgAvail.Meta.GetRelease() {
				repoEntry.Published = pkg.ID
			} else if pkg.Meta.Package.GetRelease() == pkgAvail.Meta.GetRelease() && pkgAvail.Name != pkg.ID {
				fmt.Printf(" **** DUPLICATE RELEASE NUMBER DETECTED. FAK: %s %s **** \n", pkg.ID, pkgAvail.Name)
			}
		} else {
			repoEntry.Published = pkg.ID
		}
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
	if _, err = pool.AddPackage(db, pkg, false); err != nil {
		return err
	}

	// Ensure the eopkg file is linked inside our own tree
	source := pool.GetPackagePoolPath(pkg)
	if err = LinkOrCopyFile(source, pkgTarget, false); err != nil {
		return err
	}

	return r.putEntry(db, repoEntry)
}

// AddPackage will attempt to load the local package and then add it to the
// repository via AddLocalPackage
func (r *Repository) AddPackage(db libdb.Database, pool *Pool, filename string) error {
	pkg, err := libeopkg.Open(filename)
	if err != nil {
		return err
	}

	defer pkg.Close()
	if err = pkg.ReadMetadata(); err != nil {
		return err
	}

	return r.AddLocalPackage(db, pool, pkg)
}

// GetPackageNames will traverse the buckets and find all package names as stored
// within the DB. This doesn't account for obsolete names, which should in fact
// be removed from the repo entirely.
func (r *Repository) GetPackageNames(db libdb.Database) ([]string, error) {
	var pkgIds []string
	rootBucket := db.Bucket([]byte(DatabaseBucketRepo)).Bucket([]byte(r.ID)).Bucket([]byte(DatabaseBucketPackage))

	err := rootBucket.ForEach(func(k, v []byte) error {
		entry := RepoEntry{}
		if err := rootBucket.Decode(v, &entry); err != nil {
			return err
		}
		pkgIds = append(pkgIds, entry.Name)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return pkgIds, nil
}

// GetPackages will return all package objects for a given name
func (r *Repository) GetPackages(db libdb.Database, pool *Pool, pkgName string) ([]*libeopkg.MetaPackage, error) {
	var pkgs []*libeopkg.MetaPackage

	entry, err := r.GetEntry(db, pkgName)
	if err != nil || entry == nil {
		return nil, err
	}

	for _, id := range entry.Available {
		p, err := pool.GetEntry(db, id)
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
func (r *Repository) CreateDelta(db libdb.Database, oldPkg, newPkg *libeopkg.MetaPackage) (string, error) {
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

	if err := ProduceDelta(r.deltaPath, oldPath, newPath, fullPath); err != nil {
		return "", err
	}

	return fullPath, nil
}

// HasDelta will work out if we actually have a delta already
func (r *Repository) HasDelta(db libdb.Database, pkgName, deltaPath string) (bool, error) {
	entry, err := r.GetEntry(db, pkgName)
	if err != nil {
		return false, err
	}
	for _, pkgDelta := range entry.Deltas {
		if deltaPath == pkgDelta {
			return true, nil
		}
	}
	return false, nil
}
